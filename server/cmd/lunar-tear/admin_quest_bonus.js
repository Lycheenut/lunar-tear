(() => {
  "use strict";

  const bonusTable = "m_quest_bonus";
  const labels = {
    m_quest_bonus: "关卡加成定义",
    m_quest_bonus_costume_setting_group: "共鸣服装集合",
    m_quest_bonus_weapon_group: "共鸣武器集合",
    m_quest_bonus_effect_group: "加成效果组",
    m_quest_bonus_term_group: "生效期限组",
    m_quest_bonus_ability: "能力加成",
    m_quest_bonus_exp: "经验加成",
    m_quest_bonus_drop_reward: "掉落加成"
  };
  const fieldLabels = {
    CostumeId: "服装", WeaponId: "武器（含各进化形态）", LimitBreakCountLowerLimit: "突破下限",
    QuestBonusEffectGroupId: "效果组", QuestBonusTermGroupId: "期限组（0 = 无限制）",
    QuestBonusCostumeSettingGroupId: "服装集合", QuestBonusWeaponGroupId: "武器集合",
    QuestBonusCharacterGroupId: "角色组（只读）", QuestBonusCostumeGroupId: "旧版服装组（保留值）",
    QuestBonusAllyCharacterId: "支援角色加成（只读）", SortOrder: "顺序", QuestBonusType: "类型（1 能力 / 2 经验 / 3 掉落）",
    QuestBonusEffectId: "效果定义", StartDatetime: "开始（UTC）", EndDatetime: "结束（UTC，留空为 0）",
    AbilityId: "能力 ID", Level: "等级", ExpType: "经验类型", BonusValuePermil: "加成值（千分比）",
    PossessionType: "物品类型", PossessionId: "物品 ID", AdditionalCount: "额外数量"
  };
  const references = {
    QuestBonusCharacterGroupId: "m_quest_bonus_character_group",
    QuestBonusCostumeGroupId: "m_quest_bonus_costume_group",
    QuestBonusAllyCharacterId: "m_quest_bonus_ally_character",
    QuestBonusCostumeSettingGroupId: "m_quest_bonus_costume_setting_group",
    QuestBonusWeaponGroupId: "m_quest_bonus_weapon_group",
    QuestBonusEffectGroupId: "m_quest_bonus_effect_group",
    QuestBonusTermGroupId: "m_quest_bonus_term_group"
  };
  const effectTables = { 1: "m_quest_bonus_ability", 2: "m_quest_bonus_exp", 3: "m_quest_bonus_drop_reward" };
  const difficultyName = (value) => ({ 1: "Normal", 2: "Hard", 3: "Very Hard", 4: "EX Hard" }[value] || `难度 ${value}`);
  const key = (table, id) => `${table}:${id}`;
  const number = (value) => /^\d+$/.test(String(value)) && Number.isSafeInteger(Number(value));

  class QuestBonusDraft {
    constructor(catalog) {
      this.catalog = catalog || { tables: [], quests: [], chapters: [], costumes: [], weapons: [] };
      this.tables = new Map(this.catalog.tables.map((table) => [table.name, table]));
      this.baseline = new Map();
      this.groups = new Map();
      this.assignments = new Map();
      for (const table of this.catalog.tables) {
        for (const row of table.rows || []) {
          const values = table.fields.map((field) => row.values[field.name]);
          const groupKey = key(table.name, values[0]);
          if (!this.baseline.has(groupKey)) this.baseline.set(groupKey, []);
          this.baseline.get(groupKey).push(values);
        }
      }
    }
    rows(table, id) { return this.groups.get(key(table, id))?.rows || this.baseline.get(key(table, id)) || []; }
    setGroup(table, id, rows) {
      const groupKey = key(table, id);
      const contents = (items) => items.map((row) => JSON.stringify(row)).sort().join("\n");
      if (contents(rows) === contents(this.baseline.get(groupKey) || [])) this.groups.delete(groupKey);
      else this.groups.set(groupKey, { table, groupId: Number(id), rows });
    }
    assign(quest, value) {
      if (String(value) === String(quest.bonusId)) this.assignments.delete(quest.row);
      else this.assignments.set(quest.row, { table: "m_quest", row: quest.row, field: "QuestBonusId", value: String(value) });
    }
    bonus(quest) { return this.assignments.get(quest.row)?.value ?? String(quest.bonusId); }
    payload() { return { changes: [...this.assignments.values()], questBonusGroups: [...this.groups.values()] }; }
    count() { return this.groups.size + this.assignments.size; }
    target(table, field, row) {
      if (field === "QuestBonusEffectId" && table === "m_quest_bonus_effect_group") return effectTables[row[2]];
      return references[field];
    }
    depends(table, id, target, seen = new Set()) {
      const source = key(table, id);
      if (source === target) return true;
      if (seen.has(source)) return false;
      seen.add(source);
      const spec = this.tables.get(table);
      if (!spec) return (this.catalog.readOnlyLinks?.[source] || []).some((child) => {
        const [next, nextID] = child.split(":");
        return this.depends(next, nextID, target, seen);
      });
      return this.rows(table, id).some((row) => spec.fields.some((field, i) => {
        if (field.name === "QuestBonusCostumeGroupId") return false;
        const next = i > 0 && this.target(table, field.name, row);
        return next && row[i] !== "0" && this.depends(next, row[i], target, seen);
      }));
    }
    affected(table, id) {
      return this.catalog.quests.filter((quest) => this.depends(bonusTable, this.bonus(quest), key(table, id)));
    }
    termStatus(id, now = Date.now()) {
      if (String(id) === "0") return "不限期";
      const rows = this.rows("m_quest_bonus_term_group", id);
      if (!rows.length) return "期限引用缺失";
      if (rows.some((row) => Number(row[2]) <= now && now <= Number(row[3]))) return "生效中";
      if (rows.some((row) => Number(row[2]) > now && Number(row[3]) >= Number(row[2]))) return "未开始";
      return "已过期／停用";
    }
  }

  window.QuestBonusDraft = QuestBonusDraft;
  window.createQuestBonusEditor = ({ root, onChange, localizedText, showError }) => {
    let draft = new QuestBonusDraft();
    let chapterID = "", difficulty = "", tableName = bonusTable, groupID = "", search = "";
    const selectChapter = (id) => {
      chapterID = String(id); difficulty = ""; search = "";
      tableName = bonusTable; groupID = "";
    };
    const node = (tag, text, className) => {
      const element = document.createElement(tag);
      if (text !== undefined) element.textContent = text;
      if (className) element.className = className;
      return element;
    };
    const button = (text, action) => {
      const element = node("button", text, "button ghost");
      element.type = "button";
      element.addEventListener("click", action);
      return element;
    };
    const select = (options, value, action) => {
      const element = node("select");
      for (const [id, text] of options) {
        const option = node("option", text);
        option.value = String(id);
        element.append(option);
      }
      element.value = String(value);
      element.addEventListener("change", () => action(element.value));
      return element;
    };
    const label = (text, input) => {
      const element = node("label", text);
      input.setAttribute("aria-label", text);
      element.append(input);
      return element;
    };
    const chapterName = (id) => {
      const chapter = draft.catalog.chapters.find((row) => row.values.EventQuestChapterId === String(id));
      return `${id} · ${localizedText(chapter?.titles) || (id ? "活动" : "其他关卡")}`;
    };
    const entityName = (table, id) => {
      const options = table.includes("costume") ? draft.catalog.costumes : draft.catalog.weapons;
      return localizedText(options.find((option) => String(option.id) === String(id))?.titles) || "名称不可用";
    };
    const changed = () => { onChange(); render(); };
    const openGroup = (table, id) => { tableName = table; groupID = String(id); render(); };
    const inputNumber = (value, action, list) => {
      const input = node("input");
      input.type = "text";
      input.inputMode = "numeric";
      input.value = value;
      input.pattern = "[0-9]+";
      if (list) input.setAttribute("list", list);
      input.addEventListener("change", () => {
        if (!number(input.value) || Number(input.value) > 2147483647) {
          showError("请输入 0 至 2147483647 之间的整数。");
          input.value = value;
          return;
        }
        action(String(Number(input.value)));
      });
      return input;
    };
    const groupOptions = (table) => {
      const ids = new Set();
      for (const groupKey of [...draft.baseline.keys(), ...draft.groups.keys()]) {
        const [name, id] = groupKey.split(":");
        if (name === table && draft.rows(table, id).length) ids.add(id);
      }
      return [...ids].sort((a, b) => Number(a) - Number(b));
    };
    const usageText = (quests) => {
      const chapters = [...new Set(quests.map((quest) => quest.chapterId))];
      return `${new Set(quests.map((quest) => quest.questId)).size} 个关卡 · ${chapters.map(chapterName).join("；") || "未关联活动"}`;
    };
    const renderRows = (table, rows, editable) => {
      const spec = draft.tables.get(table);
      const grid = node("table");
      const head = node("tr");
      spec.fields.forEach((field) => head.append(node("th", fieldLabels[field.name] || field.name)));
      if (editable) head.append(node("th", "状态／操作"));
      const thead = node("thead"); thead.append(head); grid.append(thead);
      const body = node("tbody"); grid.append(body);
      rows.forEach((row, index) => {
        const tr = node("tr");
        const update = (column, value) => {
          const updated = draft.rows(table, groupID).map((values) => [...values]);
          updated[index][column] = value;
          draft.setGroup(table, groupID, updated); changed();
        };
        spec.fields.forEach((field, column) => {
          const cell = node("td");
          const target = column > 0 && draft.target(table, field.name, row);
          const entity = ["CostumeId", "WeaponId"].includes(field.name);
          if (editable && column > 0 && !(target && !draft.tables.has(target))) {
            let input;
            if (field.datetime) {
              input = node("input"); input.type = "datetime-local"; input.step = "0.001";
              input.value = Number(row[column]) ? new Date(Number(row[column])).toISOString().slice(0, 23) : "";
              input.addEventListener("change", () => {
                const value = input.value ? Date.parse(`${input.value}Z`) : 0;
                if (!Number.isSafeInteger(value) || value < 0) { showError("日期无效。"); return; }
                update(column, String(value));
              });
            } else {
              input = inputNumber(row[column], (value) => update(column, value), entity ? `bonus-${field.name}` : undefined);
            }
            input.setAttribute("aria-label", `${field.name} · 行 ${index + 1}`);
            cell.append(input);
          } else cell.append(node("span", field.datetime && Number(row[column]) ? new Date(Number(row[column])).toISOString() : row[column]));
          if (entity) cell.append(node("small", entityName(table, row[column])));
          if (editable && target && draft.tables.has(target) && row[column] !== "0") {
            const exists = draft.rows(target, row[column]).length;
            cell.append(button(`${exists ? "查看" : "缺失，创建"} ${labels[target]}`, () => openGroup(target, row[column])));
          }
          tr.append(cell);
        });
        const actions = node("td");
        if (table.endsWith("costume_setting_group") || table.endsWith("weapon_group")) actions.append(node("small", draft.termStatus(row[4])));
        if (editable) actions.append(button("删除行", () => {
          draft.setGroup(table, groupID, draft.rows(table, groupID).filter((_, i) => i !== index)); changed();
        }));
        if (editable) tr.append(actions);
        body.append(tr);
      });
      const scroll = node("div", undefined, "bonus-grid"); scroll.append(grid); return scroll;
    };

    function render() {
      root.replaceChildren();
      if (!draft.catalog.tables.length) return;
      const intro = node("p", "活动 → 难度／关卡 → 加成定义 → 服装／武器集合 → 效果及期限。已过期与缺失引用也会列出；修改共享集合会影响所有引用关卡。", "bonus-note");
      root.append(intro);
      const toolbar = node("div", undefined, "bonus-toolbar");
      const matching = draft.catalog.chapters.filter((row) => [row.values.EventQuestChapterId, ...Object.values(row.titles || {})].join(" ").toLowerCase().includes(search.toLowerCase()));
      const searchInput = node("input"); searchInput.type = "search"; searchInput.value = search; searchInput.placeholder = "活动名称或 ID";
      searchInput.addEventListener("change", () => { selectChapter(""); search = searchInput.value; render(); });
      toolbar.append(label("筛选活动", searchInput));
      toolbar.append(label("活动（同名活动按 ID 区分）", select([["", "选择活动"], ...matching.map((row) => [row.values.EventQuestChapterId, chapterName(row.values.EventQuestChapterId)])], chapterID, (value) => { selectChapter(value); render(); })));
      const quests = draft.catalog.quests.filter((quest) => String(quest.chapterId) === chapterID);
      const difficulties = [...new Set(quests.map((quest) => quest.difficulty))];
      toolbar.append(label("难度", select([["", "全部难度"], ...difficulties.map((value) => [value, difficultyName(value)])], difficulty, (value) => { difficulty = value; tableName = bonusTable; groupID = ""; render(); })));
      root.append(toolbar);
      if (chapterID) {
        const chapter = draft.catalog.chapters.find((row) => row.values.EventQuestChapterId === chapterID);
        const date = (value) => Number(value) ? new Date(Number(value)).toISOString() : "0";
        root.append(node("p", `活动期限（UTC）：${date(chapter?.values.StartDatetime)} — ${date(chapter?.values.EndDatetime)}`, "bonus-note"));
        const visible = quests.filter((quest) => !difficulty || String(quest.difficulty) === difficulty);
        const mappings = new Map();
        for (const quest of visible) {
          const id = `${quest.difficulty}:${draft.bonus(quest)}`;
          if (!mappings.has(id)) mappings.set(id, []);
          mappings.get(id).push(quest);
        }
        const assignments = node("section", undefined, "bonus-card");
        assignments.append(node("h3", "关卡关联"));
        if (!visible.length) assignments.append(node("p", "该活动没有可用关卡。"));
        for (const items of mappings.values()) {
          const first = items[0], id = draft.bonus(first);
          const line = node("div", undefined, "bonus-mapping");
          line.append(node("strong", `${difficultyName(first.difficulty)} · ${items.length} 关`));
          line.append(label("加成 ID（0 = 无加成）", inputNumber(id, (value) => { items.forEach((quest) => draft.assign(quest, value)); changed(); }, "bonus-ids")));
          if (id !== "0") line.append(button(`${draft.rows(bonusTable, id).length ? "编辑" : "缺失，创建"}加成 ${id}`, () => openGroup(bonusTable, id)));
          const details = node("details"); details.append(node("summary", "逐关设置"));
          for (const quest of items) details.append(label(`关卡 ${quest.questId} · Sequence ${quest.sequenceId} / ${quest.sortOrder}`, inputNumber(draft.bonus(quest), (value) => { draft.assign(quest, value); changed(); }, "bonus-ids")));
          line.append(details); assignments.append(line);
        }
        root.append(assignments);
      }
      const editor = node("section", undefined, "bonus-card");
      const controls = node("div", undefined, "bonus-toolbar");
      controls.append(label("集合／定义", select([...draft.tables.keys()].map((name) => [name, labels[name]]), tableName, (value) => { tableName = value; groupID = ""; render(); })));
      controls.append(label("已有 ID", select([["", "选择一个 ID"], ...groupOptions(tableName).map((id) => [id, `${id} · ${draft.rows(tableName, id).length} 行`])], groupID, (id) => { groupID = id; render(); })));
      controls.append(label("打开／新建 ID", inputNumber(groupID, (id) => {
        if (id === "0") { showError("集合 ID 必须大于 0；0 保留给无加成或无期限。"); return; }
        groupID = id; render();
      })));
      editor.append(controls);
      if (groupID) {
        const rows = draft.rows(tableName, groupID);
        editor.append(node("h3", `${labels[tableName]} ${groupID} · ${rows.length} 行`));
        editor.append(node("p", `共享影响（当前草稿）：${usageText(draft.affected(tableName, groupID))}`, "bonus-note"));
        if (!rows.length) editor.append(node("p", "该 ID 当前没有定义。添加行后可补齐缺失集合；未补齐的新增引用不能保存。", "bonus-note"));
        editor.append(renderRows(tableName, rows, true));
        const actions = node("div", undefined, "bonus-toolbar");
        const single = draft.tables.get(tableName).fields.filter((field) => field.primaryKey).length === 1;
        if (!single || !rows.length) actions.append(button("添加行", () => {
          const fields = draft.tables.get(tableName).fields;
          const values = fields.map((field, i) => i === 0 ? groupID : field.name === "SortOrder" ? String(Math.max(0, ...rows.map((row) => Number(row[i]))) + 1) : "0");
          draft.setGroup(tableName, groupID, [...rows, values]); changed();
        }));
        const copyID = inputNumber("", () => {});
        copyID.setAttribute("aria-label", "复制到新 ID");
        actions.append(label("复制到新 ID", copyID), button("复制集合", () => {
          const id = String(Number(copyID.value));
          if (!number(copyID.value) || Number(id) <= 0 || Number(id) > 2147483647 || draft.baseline.has(key(tableName, id)) || draft.groups.has(key(tableName, id)) || !rows.length) {
            showError("请填写尚未使用的正整数 ID，并选择非空源集合。"); return;
          }
          draft.setGroup(tableName, id, rows.map((row) => [id, ...row.slice(1)])); groupID = id; changed();
        }));
        editor.append(actions, node("p", "复制后，请将上游引用改为新 ID。服装、武器的每个突破档位和进化形态均为独立条目；期限组 0 表示不限期，期限结束值 0 表示停用。", "bonus-note"));
      }
      root.append(editor);
      for (const [field, options] of [["CostumeId", draft.catalog.costumes], ["WeaponId", draft.catalog.weapons]]) {
        const list = node("datalist"); list.id = `bonus-${field}`;
        for (const item of options) { const option = node("option", localizedText(item.titles)); option.value = String(item.id); list.append(option); }
        root.append(list);
      }
      const ids = node("datalist"); ids.id = "bonus-ids";
      for (const id of groupOptions(bonusTable)) { const option = node("option"); option.value = id; ids.append(option); }
      root.append(ids);
    }
    return {
      load(catalog) { draft = new QuestBonusDraft(catalog); },
      reset() { draft = new QuestBonusDraft(draft.catalog); },
      selectChapter,
      count: () => draft.count(), payload: () => draft.payload(), render,
      renderPreview(container, groups) {
        for (const group of groups || []) {
          const section = node("section", undefined, "impact-group bonus-preview");
          section.append(node("h3", `${labels[group.table]} ${group.groupId} · ${group.before.length} → ${group.after.length} 行`));
          section.append(node("p", `共享影响（修改前后合计）：${usageText(group.quests || [])}`));
          const details = node("details"); details.open = true;
          details.append(node("summary", "成员及字段变化"));
          const oldRows = new Set(group.before.map((row) => JSON.stringify(row))), newRows = new Set(group.after.map((row) => JSON.stringify(row)));
          const removed = group.before.filter((row) => !newRows.has(JSON.stringify(row)));
          const added = group.after.filter((row) => !oldRows.has(JSON.stringify(row)));
          if (removed.length) details.append(node("h4", "移除／修改前"), renderRows(group.table, removed, false));
          if (added.length) details.append(node("h4", "新增／修改后"), renderRows(group.table, added, false));
          section.append(details); container.append(section);
        }
      }
    };
  };
})();
