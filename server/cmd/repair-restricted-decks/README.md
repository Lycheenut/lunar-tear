# 清空指定玩家的条件限定编队

在 `server` 目录运行，通过游戏内玩家 ID（`users.player_id`，不是登录账号或内部
`user_id`）指定唯一玩家。ID 必填；不存在或匹配多个账号时直接报错。

默认只读预览，JSON 报告输出到标准输出：

```sh
go run ./cmd/repair-restricted-decks --db db/game.db --player-id 12345 > restricted-decks-preview.json
```

核对报告后，停止游戏服务器并备份数据库，再执行：

```sh
go run ./cmd/repair-restricted-decks --db db/game.db --player-id 12345 --apply > restricted-decks-applied.json
```

完成后重新启动服务器，玩家重新登录以重新加载编队。

## 修复范围

- 清空该玩家**所有**类型 `4`（`RestrictedQuest`）及 `6`
  （`RestrictedLimitContentQuest`，包括虚光的想忆等）的编队，不按关卡或编队编号筛选。
- 保留编队编号和名称，清空三个角色槽，当前战力归零，更新变更编队的版本时间。
- 删除这些槽位的编队角色记录，卸下角色/服装、主武器、伙伴、Debris
  （`user_thought_uuid`）和换装；同时删除副武器及回忆（`user_deck_parts`）关联。
- 若异常数据中同一编队角色 UUID 也被其他类型编队引用，只解除条件限定编队的引用，
  保留其他编队所需的角色及装备关联，并在报告中列出 UUID。
- 背包中的角色、武器、Debris、伙伴和回忆全部保留。普通/PvP/多人/讨伐战编队、
  三队组合、历史最高战力、关卡进度及通关产生的出战限制记录
  （`user_deck_limit_content_restricted`）保持不变；此工具只清空编队配置。

`Decks` 列出将变更的编队及原角色槽 UUID、战力；`RemovedDeckCharacters`、
`RemovedSubWeapons`、`RemovedParts` 是将删除的**编队关联记录数**，不是背包物品数量。
`SharedDeckCharactersKept` 列出保留给其他类型编队使用的角色 UUID。
预览时 `Applied` 为 `false`，成功提交后为 `true`。

所有修改在单个事务中提交，失败全部回滚。可重复执行：已经清空且战力为零的编队
不会再次写入；没有需要修复的编队时，`Decks` 为空、删除计数为零。
工具不创建数据库、不执行迁移、不读取主数据，也不触发奖励补发。

```sh
go build -o bin/repair-restricted-decks ./cmd/repair-restricted-decks
go test ./cmd/repair-restricted-decks
```
