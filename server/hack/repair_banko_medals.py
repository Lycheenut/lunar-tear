#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""单文件修复「記録：蛮鼓の砦」(活动 513) 的掉落道具和掉落加成。

运行环境：Python 3.8+。仅需上传此文件，不需要 Go、项目代码或 schemas.json。
安装依赖：python3 -m pip install pycryptodome msgpack lz4
只读预览：python3 repair_banko_medals.py --input 20240404193219.bin.e
生成文件：python3 repair_banko_medals.py --input 20240404193219.bin.e --output fixed.bin.e

将关卡引用的消耗品 63/64/65（銅/銀/金）统一为 29「蛮鼓の砦メダル」。
保留掉落数量和抽选行；同一加成组的等值奖章加成合并为一份（+5/+5/+5 -> +5）。
不同数值的重复加成或跨活动共用的待修改配置会报错，不写出文件。
默认仅预览；输出必须是新文件，原主数据和已有输出永远不会被覆盖。
结果会重新解密验证；已修复文件再次执行不会产生额外修改。

确认结果后，备份并替换服务器使用的主数据，再重载主数据或重启服务。
本脚本不会自动安装或重载，也不会修改玩家背包、商店、活动时间或 JSON 导出。
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import sys

try:
    import lz4.block
    import msgpack
    from Crypto.Cipher import AES
    from Crypto.Util.Padding import pad, unpad
except ImportError as exc:
    raise SystemExit(
        "Missing dependency: {}\nInstall: python3 -m pip install pycryptodome msgpack lz4".format(exc)
    ) from exc


KEY = bytes.fromhex("36436230313332314545356536624265")
IV = bytes.fromhex("45666341656634434165356536446141")
CHAPTER = 513
MEDAL = 29
WRONG_MEDALS = {63, 64, 65}
DROP = "m_battle_drop_reward"
BONUS_DROP = "m_quest_bonus_drop_reward"
EFFECT = "m_quest_bonus_effect_group"

# Serialized MasterMemory column positions, embedded so no schema file is needed.
COLUMNS = {
    "m_consumable_item": (0, 1, 6, 7),
    "m_event_quest_chapter": (0, 1, 3, 7),
    "m_event_quest_sequence_group": (0, 2),
    "m_event_quest_sequence": (0, 2),
    "m_quest": (0, 8, 19),
    "m_quest_pickup_reward_group": (0, 2),
    DROP: (0, 1, 2, 3),
    "m_quest_bonus": (0, 1, 2, 3, 4, 5),
    "m_quest_bonus_character_group": (0, 2),
    "m_quest_bonus_costume_group": (0, 2),
    "m_quest_bonus_weapon_group": (0, 3),
    "m_quest_bonus_costume_setting_group": (0, 3),
    "m_quest_bonus_ally_character": (0, 1),
    EFFECT: (0, 1, 2, 3),
    BONUS_DROP: (0, 1, 2, 3),
}


def unpacker(raw):
    reader = msgpack.Unpacker(raw=False, strict_map_key=False, max_buffer_size=len(raw) + 1)
    reader.feed(raw)
    return reader


def decode_table(blob):
    obj = msgpack.unpackb(blob, raw=False, strict_map_key=False)
    if isinstance(obj, msgpack.ExtType):
        if obj.code != 99:
            raise ValueError("Unsupported table compression code: {}".format(obj.code))
        reader = unpacker(obj.data)
        size = reader.unpack()
        if type(size) is not int or size <= 0:
            raise ValueError("Invalid LZ4 uncompressed size")
        raw = lz4.block.decompress(obj.data[reader.tell():], uncompressed_size=size)
        if len(raw) != size:
            raise ValueError("Incorrect LZ4 uncompressed size")
        return raw, True
    if not isinstance(obj, list):
        raise ValueError("Expected an array of table rows")
    return blob, False


def encode_table(raw, compressed):
    if not compressed:
        return raw
    payload = b"\xd2" + struct.pack(">i", len(raw))
    payload += lz4.block.compress(raw, store_size=False)
    return msgpack.packb(msgpack.ExtType(99, payload), use_bin_type=True)


def patch_table(raw, edits, removed):
    """Patch item cells/delete duplicate rows without re-encoding other values."""
    reader = unpacker(raw)
    count = reader.read_array_header()
    header = raw[:reader.tell()]
    if (set(edits) | removed) - set(range(count)):
        raise ValueError("Edit points outside the table")
    if removed:
        header = msgpack.Packer().pack_array_header(count - len(removed))
    pieces = [header]
    for index in range(count):
        start = reader.tell()
        if index in edits:
            columns = reader.read_array_header()
            if columns < 3:
                raise ValueError("Reward row is missing its item column")
            for col in range(columns):
                cell_start = reader.tell()
                reader.skip()
                if col == 2:
                    left, right = cell_start, reader.tell()
            # The client expects the edited ID to retain its int32 schema width.
            row = raw[start:left] + b"\xd2" + struct.pack(">i", edits[index]) + raw[right:reader.tell()]
        else:
            reader.skip()
            row = raw[start:reader.tell()]
        if index not in removed:
            pieces.append(row)
    if reader.tell() != len(raw):
        raise ValueError("Unexpected trailing table data")
    return b"".join(pieces)


class MasterData:
    def __init__(self, encrypted):
        self.encrypted = encrypted
        clear = unpad(AES.new(KEY, AES.MODE_CBC, IV).decrypt(encrypted), AES.block_size)
        reader = unpacker(clear)
        self.toc = reader.unpack()
        self.blob = clear[reader.tell():]
        if not isinstance(self.toc, dict):
            raise ValueError("Invalid MasterMemory header")
        for name, bounds in self.toc.items():
            if (not isinstance(name, str) or not isinstance(bounds, list) or len(bounds) != 2
                    or any(type(n) is not int or n < 0 for n in bounds)
                    or sum(bounds) > len(self.blob)):
                raise ValueError("Invalid table range: {}".format(name))

    def table_blob(self, name):
        offset, length = self.toc[name]
        return self.blob[offset:offset + length]

    def tables(self):
        data = {}
        for name, fields in COLUMNS.items():
            if name not in self.toc:
                raise ValueError("Missing table: " + name)
            raw, _ = decode_table(self.table_blob(name))
            rows = msgpack.unpackb(raw, raw=False, strict_map_key=False)
            if not isinstance(rows, list):
                raise ValueError("Invalid row array: " + name)
            for index, row in enumerate(rows):
                if (not isinstance(row, list) or len(row) <= max(fields)
                        or any(type(row[col]) is not int or not 0 <= row[col] <= 0x7fffffff for col in fields)):
                    raise ValueError("Invalid schema: {} row {}".format(name, index))
            data[name] = rows
        return data

    def rebuild(self, edits, removed):
        if not edits and not removed:
            return self.encrypted
        packer = msgpack.Packer(use_bin_type=True)
        header = [packer.pack_map_header(len(self.toc))]
        chunks, offset = [], 0
        # Preserve the serialized table order and signed int32 TOC fields.
        for name in sorted(self.toc, key=lambda key: self.toc[key][0]):
            blob = self.table_blob(name)
            deletions = removed if name == EFFECT else set()
            if name in edits or deletions:
                raw, compressed = decode_table(blob)
                blob = encode_table(patch_table(raw, edits.get(name, {}), deletions), compressed)
            header.append(packer.pack(name) + b"\x92\xd2" + struct.pack(">i", offset)
                          + b"\xd2" + struct.pack(">i", len(blob)))
            chunks.append(blob)
            offset += len(blob)
        clear = b"".join(header + chunks)
        return AES.new(KEY, AES.MODE_CBC, IV).encrypt(pad(clear, AES.block_size))


def references(data, table, sources, column):
    return {row[column] for row in data[table] if row[0] in sources and row[column] != 0}


def target_quests(data):
    if not any(row[0] == MEDAL and row[1] == 110 and row[6:8] == [110, 29]
               for row in data["m_consumable_item"]):
        raise ValueError("Expected medal (consumable item 29) is missing")
    chapters = [row for row in data["m_event_quest_chapter"] if row[0] == CHAPTER]
    if any(row[1] != 1 or row[3] != 513 for row in chapters):
        raise ValueError("Chapter 513 is not the expected Record event")
    sequences = references(data, "m_event_quest_sequence_group", {row[7] for row in chapters}, 2)
    quests = references(data, "m_event_quest_sequence", sequences, 2)
    if not quests:
        raise ValueError("Chapter 513 has no quests")
    missing = quests - {row[0] for row in data["m_quest"]}
    if missing:
        raise ValueError("Missing event quests: {}".format(sorted(missing)))
    other_groups = {row[7] for row in data["m_event_quest_chapter"] if row[0] != CHAPTER}
    other_sequences = references(data, "m_event_quest_sequence_group", other_groups, 2)
    shared = quests & references(data, "m_event_quest_sequence", other_sequences, 2)
    if shared:
        raise ValueError("Quests shared with another chapter: {}".format(sorted(shared)))
    return quests


def bonus_effects(data, quests):
    bonuses = references(data, "m_quest", quests, 19)
    effects = set()
    for table, bonus_col, effect_col in (
        ("m_quest_bonus_character_group", 1, 2),
        ("m_quest_bonus_costume_group", 2, 2),
        ("m_quest_bonus_weapon_group", 3, 3),
        ("m_quest_bonus_costume_setting_group", 4, 3),
        ("m_quest_bonus_ally_character", 5, 1),
    ):
        groups = references(data, "m_quest_bonus", bonuses, bonus_col)
        effects.update(references(data, table, groups, effect_col))
    return effects


def bonus_drops(data, effects):
    return {row[3] for row in data[EFFECT] if row[0] in effects and row[2] == 3}


def plan_repair(data):
    targets = target_quests(data)
    others = {row[0] for row in data["m_quest"]} - targets
    pickups = references(data, "m_quest", targets, 8)
    other_pickups = references(data, "m_quest", others, 8)
    drops = references(data, "m_quest_pickup_reward_group", pickups, 2)
    other_drops = references(data, "m_quest_pickup_reward_group", other_pickups, 2)
    effects, other_effects = bonus_effects(data, targets), bonus_effects(data, others)
    edits, removed = {}, set()
    report = {"chapterId": CHAPTER, "questIds": sorted(targets),
              "currencyChanges": [], "duplicateBonusEffectsRemoved": []}
    for table, selected, outside in (
        (DROP, drops, other_drops),
        (BONUS_DROP, bonus_drops(data, effects), bonus_drops(data, other_effects)),
    ):
        for index, row in enumerate(data[table]):
            identity, possession_type, item = row[:3]
            if identity not in selected or possession_type != 6 or item not in WRONG_MEDALS:
                continue
            if identity in outside:
                raise ValueError("{} ID {} is shared with quests outside chapter 513".format(table, identity))
            edits.setdefault(table, {})[index] = MEDAL
            report["currencyChanges"].append({"table": table, "id": identity, "from": item, "to": MEDAL})
    drop_by_id = {row[0]: row for row in data[BONUS_DROP]}
    counts = {}
    for index, row in enumerate(data[EFFECT]):
        if len(row) != 4:
            raise ValueError("Unexpected effect group schema")
        group, order, bonus_type, identity = row
        if group not in effects or bonus_type != 3:
            continue
        if identity not in drop_by_id:
            raise ValueError("Effect group {} references missing bonus drop {}".format(group, identity))
        _, possession_type, item, count = drop_by_id[identity]
        if possession_type != 6 or item not in WRONG_MEDALS | {MEDAL}:
            continue
        if group in counts:
            if counts[group] != count:
                raise ValueError("Effect group {} has conflicting medal bonuses: {} and {}".format(group, counts[group], count))
            if group in other_effects:
                raise ValueError("Effect group {} is shared with quests outside chapter 513".format(group))
            removed.add(index)
            report["duplicateBonusEffectsRemoved"].append({"groupId": group, "sortOrder": order, "effectId": identity})
        else:
            counts[group] = count
    return report, edits, removed


def main(argv=None):
    parser = argparse.ArgumentParser(description="Repair chapter 513 medals (63/64/65 -> 29), keeping one equal-valued bonus per group.")
    parser.add_argument("--input", type=Path, required=True, help="Source .bin.e master data (never modified)")
    parser.add_argument("--output", type=Path, help="Write a NEW file; omitted means read-only preview")
    args = parser.parse_args(argv)
    source = MasterData(args.input.read_bytes())
    report, edits, removed = plan_repair(source.tables())
    print(json.dumps(report, ensure_ascii=False, indent=2))
    print("Currency edits: {}; duplicate bonus effects removed: {}".format(len(report["currencyChanges"]), len(removed)))
    if args.output is None:
        print("Preview only; no files written.")
        return 0
    candidate = source.rebuild(edits, removed)
    _, remaining_edits, remaining_removed = plan_repair(MasterData(candidate).tables())
    if remaining_edits or remaining_removed:
        raise ValueError("Verification failed: candidate still needs repairs")
    # Exclusive creation refuses both source overwrite and existing output files.
    destination = args.output.open("xb")
    try:
        with destination:
            destination.write(candidate)
            destination.flush()
            os.fsync(destination.fileno())
    except BaseException:
        args.output.unlink(missing_ok=True)
        raise
    print("Verified output: {}\nSHA-256: {}".format(args.output, hashlib.sha256(candidate).hexdigest()))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, EOFError, RuntimeError, msgpack.exceptions.UnpackException) as exc:
        print("error: {}".format(exc), file=sys.stderr)
        sys.exit(1)
