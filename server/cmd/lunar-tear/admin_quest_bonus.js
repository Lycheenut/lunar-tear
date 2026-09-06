(() => {
  "use strict";
  const bonusTable = "m_quest_bonus", costumeTable = "m_quest_bonus_costume_setting_group", weaponTable = "m_quest_bonus_weapon_group";
  const effectTable = "m_quest_bonus_effect_group", dropTable = "m_quest_bonus_drop_reward";
  const text = titles => Object.values(titles || {}).join(" ");
  const unique = values => [...new Set(values)];

  class QuestBonusDraft {
    constructor(catalog) {
      this.catalog = catalog || { tables: [], chapters: [], quests: [], costumes: [], weapons: [], medals: [] };
      this.groups = new Map(); this.bonuses = new Map(); this.selections = new Map(); this.membersCache = new Map(); this.profilesCache = new Map();
      this.costumes = new Map(this.catalog.costumes.map(item => [String(item.id), item]));
      this.weapons = new Map(this.catalog.weapons.map(item => [String(item.id), item]));
      this.medals = new Map((this.catalog.medals || []).map(item => [String(item.id), item]));
      for (const table of this.catalog.tables) for (const row of table.rows || []) {
        const id = row.values[table.fields[0].name], key = `${table.name}:${id}`;
        if (!this.groups.has(key)) this.groups.set(key, []);
        this.groups.get(key).push(row.values);
        if (table.name === bonusTable) this.bonuses.set(String(id), row.values);
      }
    }
    rows(table, id) { return this.groups.get(`${table}:${id}`) || []; }
    quests(chapterID) { return this.catalog.quests.filter(q => String(q.chapterId) === String(chapterID)); }
    currentIDs(chapterID) { return unique(this.quests(chapterID).map(q => String(q.bonusId))); }
    family(id) { return String(this.weapons.get(String(id))?.evolutionGroupId || id); }
    order(id) { return this.weapons.get(String(id))?.evolutionOrder || 0; }
    members(ids) {
      const cacheKey = [...ids].sort().join(",");
      if (this.membersCache.has(cacheKey)) return this.membersCache.get(cacheKey);
      const result = new Map();
      for (const id of ids) {
        const bonus = this.bonuses.get(String(id)); if (!bonus) continue;
        for (const [kind, table, groupField, itemField, options] of [
          ["服装", costumeTable, "QuestBonusCostumeSettingGroupId", "CostumeId", this.costumes],
          ["武器", weaponTable, "QuestBonusWeaponGroupId", "WeaponId", this.weapons]
        ]) for (const row of this.rows(table, bonus[groupField])) {
          const itemID = String(row[itemField]), key = `${kind}:${kind === "武器" ? this.family(itemID) : itemID}`;
          if (!result.has(key)) result.set(key, { key, kind, itemID, titles: options.get(itemID)?.titles, itemIDs: [] });
          result.get(key).itemIDs.push(itemID);
        }
      }
      for (const member of result.values()) member.itemIDs = unique(member.itemIDs).sort((a, b) => this.order(a) - this.order(b) || Number(a) - Number(b));
      this.membersCache.set(cacheKey, result); return result;
    }
    weaponRows(bonusID) { return this.rows(weaponTable, this.bonuses.get(String(bonusID))?.QuestBonusWeaponGroupId); }
    rewards(effectID) {
      return this.rows(effectTable, effectID).flatMap(effect => effect.QuestBonusType === "3" ? this.rows(dropTable, effect.QuestBonusEffectId).map(drop => ({ possessionType: Number(drop.PossessionType), possessionId: Number(drop.PossessionId), count: Number(drop.AdditionalCount) })) : []);
    }
    targetGroups(chapterID) {
      const groups = new Map();
      for (const q of this.quests(chapterID)) {
        const key = `${q.bonusId}:${[...(q.medalIds || [])].sort((a, b) => a - b).join(",")}`;
        if (!groups.has(key)) groups.set(key, { key, bonusID: String(q.bonusId), quests: [], medalIDs: q.medalIds || [] });
        groups.get(key).quests.push(q);
      }
      return [...groups.values()];
    }
    profiles(chapterID) {
      const key = String(chapterID); if (this.profilesCache.has(key)) return this.profilesCache.get(key);
      const ids = this.currentIDs(key).filter(id => id !== "0"), merged = new Map();
      const members = [...this.members(ids).values()].filter(m => m.kind === "武器");
      for (const member of members) {
        const family = this.family(member.itemID);
        const phases = ids.map(id => this.weaponRows(id).filter(row => this.family(row.WeaponId) === family).map(row => [this.order(row.WeaponId), Number(row.LimitBreakCountLowerLimit), this.rewards(row.QuestBonusEffectGroupId).sort((a, b) => a.possessionId - b.possessionId)]).sort((a, b) => a[0] - b[0] || a[1] - b[1]));
        // Profiles cover the complete activity, including every phase and evolution form.
        if (phases.some(rows => !rows.length)) continue;
        const signature = JSON.stringify(phases);
        if (!merged.has(signature)) merged.set(signature, { id: member.itemID, members: [], medalIDs: unique(phases.flat(1).flatMap(row => row[2].map(reward => reward.possessionId))) });
        merged.get(signature).members.push(member);
      }
      const result = [...merged.values()]; this.profilesCache.set(key, result); return result;
    }
    templateRows(bonusID, weaponID, templateID) {
      const rows = this.weaponRows(bonusID).filter(row => this.family(row.WeaponId) === this.family(templateID) && this.order(row.WeaponId) <= this.order(weaponID));
      const order = Math.max(-1, ...rows.map(row => this.order(row.WeaponId)));
      return rows.filter(row => this.order(row.WeaponId) === order);
    }
    replace(chapterID, bonusID) {
      const key = String(chapterID);
      if (bonusID === "") { this.selections.delete(key); return; }
      if (!this.quests(key).length) throw new Error("该活动没有可还原的关卡。");
      if (!this.bonuses.has(String(bonusID))) throw new Error("请选择一个已有加成条目。");
      const ids = new Set(this.quests(key).map(q => q.questId));
      if (this.catalog.quests.some(q => String(q.chapterId) !== key && ids.has(q.questId))) throw new Error("此活动与其他活动共用关卡，需先分离关卡引用才能使用独立活动期限。");
      const selected = { sourceBonusID: String(bonusID), ruleChapterID: this.selections.get(key)?.ruleChapterID || key, choices: {}, groups: {}, currencies: {} };
      this.selections.set(key, selected); this.setReference(key, selected.ruleChapterID);
    }
    setReference(chapterID, ruleChapterID) {
      const selected = this.selections.get(String(chapterID));
      selected.ruleChapterID = String(ruleChapterID); selected.choices = {}; selected.groups = {}; selected.currencies = {};
      const profiles = this.profiles(ruleChapterID);
      for (const member of this.members([selected.sourceBonusID]).values()) if (member.kind === "武器") {
        const profile = profiles.find(p => p.members.some(m => m.key === member.key));
        if (profile) selected.choices[member.key] = profile.id;
      }
    }
    count() { return this.selections.size; }
    payload() {
      const restores = [];
      for (const [chapterID, selected] of this.selections) {
        const source = [...this.members([selected.sourceBonusID]).values()].filter(m => m.kind === "武器");
        const external = selected.ruleChapterID !== chapterID;
        const groups = this.targetGroups(chapterID);
        if (!external && groups.some(group => group.bonusID === "0")) throw new Error(`活动 ${chapterID} 缺少现有加成，请选择规则参考活动并设置对应关系。`);
        if (external && groups.some(group => !selected.groups[group.key])) throw new Error(`活动 ${chapterID} 尚未完成关卡分组对应。`);
        const weapons = source.map(member => {
          const template = selected.choices[member.key];
          if (!template) throw new Error(`活动 ${chapterID}：${text(member.titles) || member.itemID} 尚未选择武器规则。`);
          const profile = this.profiles(selected.ruleChapterID).find(p => p.id === template);
          if (!profile) throw new Error("所选武器规则已失效，请重新选择。");
          for (const group of groups) for (const id of member.itemIDs) if (!this.templateRows(external ? selected.groups[group.key] : group.bonusID, id, template).length) throw new Error(`武器 ${id} 的进化形态缺少适用规则。`);
          if (external && profile.medalIDs.some(id => !selected.currencies[id])) throw new Error(`活动 ${chapterID} 尚未完成奖章对应。`);
          return { weaponId: Number(member.itemID), templateWeaponId: Number(template) };
        });
        const input = { chapterId: Number(chapterID), sourceBonusId: Number(selected.sourceBonusID), ruleChapterId: Number(selected.ruleChapterID), weapons };
        if (external) {
          input.groups = groups.map(group => ({ questIds: group.quests.map(q => q.questId), ruleBonusId: Number(selected.groups[group.key]) }));
          input.currencies = Object.entries(selected.currencies).filter(([, value]) => value).map(([from, to]) => ({ fromId: Number(from), toId: Number(to) }));
        }
        restores.push(input);
      }
      return { changes: [], questBonusGroups: [], questBonusRestores: restores };
    }
  }
  window.QuestBonusDraft = QuestBonusDraft;

  window.createQuestBonusEditor = ({ root, onChange, localizedText, showError, formatDatetime }) => {
    let draft = new QuestBonusDraft(), chapterID = "", phaseIndex = 0, limitBreak = 4;
    const node = (tag, content, className) => { const el = document.createElement(tag); if (content !== undefined) el.textContent = content; if (className) el.className = className; return el; };
    const button = (content, action, className = "button ghost") => { const el = node("button", content, className); el.type = "button"; el.addEventListener("click", action); return el; };
    const title = member => localizedText(member?.titles) || member?.itemID || String(member?.id || "");
    const chapter = id => draft.catalog.chapters.find(row => row.values.EventQuestChapterId === String(id));
    const chapterTitle = id => localizedText(chapter(id)?.titles) || `活动 ${id}`;
    const date = formatDatetime || (value => Number(value) ? new Date(Number(value)).toLocaleString("zh-CN", { hour12: false }) : "停用");
    const medal = id => title(draft.medals.get(String(id))) || `物品 ${id}`;
    const shortMedal = id => {
      const name = medal(id), suffix = name.match(/メダル[：:](銅|銀|金)$/);
      return suffix ? `${suffix[1]}メダル` : name.replace(/^.*?(?=金メダル|銀メダル|銅メダル|金奖章|银奖章|铜奖章)/, "");
    };
    const range = (start, end) => `${date(start)} — ${date(end)}`;
    const select = (label, entries, value, action, searchable = true) => {
      const el = node("select"); el.setAttribute("aria-label", label); if (searchable) el.dataset.searchable = "true";
      for (const entry of entries) {
        const option = node("option", entry.label); option.value = entry.value;
        if (entry.search) option.dataset.search = entry.search;
        if (entry.group) option.dataset.searchGroup = entry.group;
        el.append(option);
      }
      el.value = value; el.addEventListener("change", () => { try { action(el.value); } catch (error) { showError(error.message); } }); return el;
    };
    const field = (label, control) => { const el = node("label", undefined, "bonus-field"); el.append(node("span", label), control); return el; };
    const groupLabel = group => {
      const difficulties = unique(group.quests.map(q => q.difficulty));
      const firstEX = Math.min(...draft.quests(group.quests[0]?.chapterId).filter(q => q.difficulty === 4).map(q => q.sortOrder));
      const stages = difficulties.length > 1 ? (difficulties.includes(4) ? "常规＋EX Hard 1" : "常规关卡") : difficulties[0] === 4 ? `EX Hard ${group.quests.map(q => q.sortOrder - firstEX + 1).join(" / ")}` : ["", "Normal", "Hard", "Very Hard"][difficulties[0]] || `难度 ${difficulties[0]}`;
      return `${stages} · ${group.quests.length} 关 · ${group.medalIDs.map(shortMedal).join("＋") || "无奖章"}`;
    };
    function mappingDialog(selected) {
      const dialog = node("dialog", undefined, "bonus-mapping-dialog");
      dialog.append(node("h2", "关卡与奖章对应"), node("p", "按实际掉落选择对应关系，数量与突破档位沿用参考规则。", "bonus-note"));
      const rules = draft.currentIDs(selected.ruleChapterID).filter(id => id !== "0").map(id => {
        const groups = draft.targetGroups(selected.ruleChapterID).filter(group => group.bonusID === id);
        return { value: id, label: `${id} · ${groups.map(groupLabel).join("；")}` };
      });
      const grid = node("div", undefined, "bonus-mapping-grid");
      for (const group of draft.targetGroups(chapterID)) grid.append(field(groupLabel(group), select(`参考分组 ${group.key}`, [{ value: "", label: "选择参考关卡分组…" }, ...rules], selected.groups[group.key] || "", value => { selected.groups[group.key] = value; onChange(); })));
      dialog.append(node("h3", "目标关卡 → 参考分组"), grid, node("h3", "参考奖章 → 目标奖章"));
      const currencies = node("div", undefined, "bonus-mapping-grid");
      const available = unique(draft.quests(chapterID).flatMap(q => q.medalIds || []));
      const required = unique(draft.profiles(selected.ruleChapterID).filter(p => Object.values(selected.choices).includes(p.id)).flatMap(p => p.medalIDs));
      for (const id of required) currencies.append(field(`${medal(id)} · ${id}`, select(`目标奖章 ${id}`, [{ value: "", label: "选择本活动实际掉落的奖章…" }, ...available.map(value => ({ value: String(value), label: `${medal(value)} · ${value}` }))], selected.currencies[id] || "", value => { selected.currencies[id] = value; onChange(); })));
      if (!required.length) currencies.append(node("p", "先选择武器规则，再设置这些规则使用的奖章。", "bonus-note"));
      dialog.append(currencies, button("完成", () => dialog.close()));
      dialog.addEventListener("close", () => { dialog.remove(); render(); }); document.body.append(dialog); dialog.showModal();
    }
    function rewardsAt(bonusID, weaponID, templateID, selected) {
      const rows = draft.templateRows(bonusID, weaponID, templateID).filter(row => Number(row.LimitBreakCountLowerLimit) <= limitBreak).sort((a, b) => Number(b.LimitBreakCountLowerLimit) - Number(a.LimitBreakCountLowerLimit));
      if (!rows.length) return "无适用规则";
      return draft.rewards(rows[0].QuestBonusEffectGroupId).map(reward => `${shortMedal(selected?.currencies[reward.possessionId] || reward.possessionId)} +${reward.count}`).join(" / ") || "无掉落加成";
    }
    function roster(ids, label, selected, phase) {
      const panel = node("section", undefined, "bonus-roster");
      const members = [...draft.members(ids).values()], costumes = members.filter(m => m.kind === "服装"), weapons = members.filter(m => m.kind === "武器");
      panel.append(node("h3", `${label} · ${costumes.length} 套服装 / ${weapons.length} 种武器`));
      const names = node("div", undefined, "bonus-costume-list");
      for (const member of costumes) { const chip = node("span", title(member)); chip.title = `${title(member)} · ${member.itemIDs.join(" / ")}`; names.append(chip); }
      if (!costumes.length) names.append(node("span", "无共鸣服装", "bonus-note"));
      panel.append(names);
      if (!weapons.length) panel.append(node("p", "无共鸣武器", "bonus-note"));
      const profiles = selected ? draft.profiles(selected.ruleChapterID) : [];
      for (const member of weapons) {
        const row = node("div", undefined, "bonus-weapon-row");
        const name = node("strong", title(member)); name.title = `${title(member)} · ${member.itemIDs.join(" / ")}`; row.append(name);
        if (selected) {
          const picker = select(`武器规则 ${member.itemID}`, [{ value: "", label: "选择完整武器规则…" }, ...profiles.map((profile, index) => ({ value: profile.id, label: `规则 ${index + 1} · ${profile.members.map(title).join(" / ")}`, search: profile.members.flatMap(m => [...m.itemIDs, text(m.titles)]).join(" ") }))], selected.choices[member.key] || "", value => { selected.choices[member.key] = value; onChange(); render(); });
          row.append(picker);
          const template = selected.choices[member.key], ruleID = selected.ruleChapterID === chapterID ? phase?.bonusID : selected.groups[phase?.key];
          const values = template && ruleID ? unique(member.itemIDs.map(id => rewardsAt(ruleID, id, template, selected))) : ["待选择规则或分组对应"];
          const info = node("small", values.join("；"), template && ruleID ? "" : "bonus-warning"); info.title = values.join("；"); row.append(info);
        } else {
          const values = phase ? unique(member.itemIDs.map(id => rewardsAt(phase.bonusID, id, id))) : [];
          const info = node("small", values.join("；")); info.title = values.join("；"); row.append(info);
        }
        panel.append(row);
      }
      return panel;
    }
    function render() {
      root.replaceChildren(); if (!draft.catalog.chapters.length) return;
      const sidebar = node("aside", undefined, "bonus-events");
      sidebar.append(field("目标活动", select("目标活动", [{ value: "", label: "搜索活动标题或 ID…" }, ...draft.catalog.chapters.map(row => ({ value: row.values.EventQuestChapterId, label: `${chapterTitle(row.values.EventQuestChapterId)} · ${row.values.EventQuestChapterId}`, search: text(row.titles) }))], chapterID, value => { chapterID = value; phaseIndex = 0; render(); })));
      sidebar.append(node("p", "导入完整历史名单，服装效果沿用来源，武器按本期奖章规则生效。", "bonus-note"));
      if (chapter(chapterID)) sidebar.append(node("p", `全部共鸣跟随活动\n开始 ${date(chapter(chapterID).values.StartDatetime)}\n结束 ${date(chapter(chapterID).values.EndDatetime)}`, "bonus-note bonus-activity-dates"));
      const pending = node("div", undefined, "bonus-event-list");
      for (const id of unique([chapterID, ...draft.selections.keys()]).filter(Boolean)) {
        const item = button("", () => { chapterID = id; phaseIndex = 0; render(); }, `bonus-event${id === chapterID ? " selected" : ""}`);
        item.append(node("strong", chapterTitle(id)), node("small", `${id} · ${draft.selections.has(id) ? "待应用" : "查看中"}`)); item.title = chapterTitle(id); pending.append(item);
      }
      sidebar.append(pending); root.append(sidebar);
      const main = node("section", undefined, "bonus-replacement"); root.append(main);
      if (!chapter(chapterID)) { main.append(node("h2", "QuestBonus"), node("p", "选择活动和历史名单，再为新增武器指定规则。", "bonus-note")); return; }
      const currentIDs = draft.currentIDs(chapterID), selected = draft.selections.get(chapterID), groups = draft.targetGroups(chapterID);
      phaseIndex = Math.min(phaseIndex, Math.max(0, groups.length - 1)); const phase = groups[phaseIndex];
      const heading = node("div", undefined, "bonus-replacement-heading"); heading.append(node("h2", chapterTitle(chapterID)), node("span", `${chapterID} · ${draft.quests(chapterID).length} 关卡`, "row-badge")); main.append(heading);
      const pickers = node("div", undefined, "bonus-source-picker");
      const sources = [...draft.bonuses].sort(([a], [b]) => Number(a) - Number(b)).map(([id, bonus]) => {
        const members = [...draft.members([id]).values()];
        return { value: id, label: `${id} · 服装组 ${bonus.QuestBonusCostumeSettingGroupId} / 武器组 ${bonus.QuestBonusWeaponGroupId} · ${members.slice(0, 2).map(title).join(" / ") || "空名单"}`, search: [...Object.values(bonus), ...members.flatMap(m => [text(m.titles), ...m.itemIDs])].join(" "), group: Object.entries(bonus).filter(([name]) => name !== "QuestBonusId").map(([, value]) => value).join(":") };
      });
      pickers.append(field("历史名单来源", select("历史名单来源", [{ value: "", label: "搜索加成 ID、组 ID 或服装／武器名称…" }, ...sources], selected?.sourceBonusID || "", value => { draft.replace(chapterID, value); onChange(); render(); })));
      const reference = select("规则参考活动", draft.catalog.chapters.filter(row => row.values.EventQuestChapterId === chapterID || draft.currentIDs(row.values.EventQuestChapterId).some(id => id !== "0")).map(row => ({ value: row.values.EventQuestChapterId, label: `${row.values.EventQuestChapterId === chapterID ? "本活动 · " : ""}${chapterTitle(row.values.EventQuestChapterId)} · ${row.values.EventQuestChapterId}` })), selected?.ruleChapterID || chapterID, value => { draft.setReference(chapterID, value); onChange(); render(); });
      reference.disabled = !selected; pickers.append(field("武器规则参考", reference)); main.append(pickers);
      const controls = node("div", undefined, "bonus-phase-controls");
      controls.append(field("关卡预览", select("预览关卡分组", groups.map((group, index) => ({ value: String(index), label: groupLabel(group) })), String(phaseIndex), value => { phaseIndex = Number(value); render(); })), field("突破", select("预览突破档位", [0, 1, 2, 3, 4].map(value => ({ value: String(value), label: `${value} 突破` })), String(limitBreak), value => { limitBreak = Number(value); render(); }, false)));
      if (selected && selected.ruleChapterID !== chapterID) controls.append(button("关卡／奖章对应", () => mappingDialog(selected)));
      main.append(controls);
      const scroll = node("div", undefined, "bonus-diff-scroll"), comparison = node("div", undefined, "bonus-comparison");
      comparison.append(roster(currentIDs, "当前名单", null, phase));
      if (selected) comparison.append(roster([selected.sourceBonusID], "还原名单", selected, phase));
      else comparison.append(node("p", "选择历史来源后显示完整名单与武器规则。", "bonus-note"));
      scroll.append(comparison); main.append(scroll);
      const note = node("div", undefined, "bonus-replacement-note");
      note.append(node("p", "每项武器规则涵盖全部关卡分组、突破档位与进化形态；上方切换仅用于预览。"));
      if (selected) {
        try { draft.payload(); note.append(node("p", "配置完整，点击下方“重建并应用”查看保存预览。", "bonus-added")); }
        catch (error) { note.append(node("p", error.message, "bonus-warning")); }
        note.append(button("撤销本活动", () => { draft.replace(chapterID, ""); onChange(); render(); }));
      }
      main.append(note);
    }
    const handlesRecord = record => record.table === "m_quest" && record.changes?.some(change => change.field === "QuestBonusId");
    return {
      load(catalog) { draft = new QuestBonusDraft(catalog); }, reset() { draft.selections.clear(); },
      selectChapter(id) { chapterID = String(id); phaseIndex = 0; }, count: () => draft.count(), payload: () => draft.payload(), render, handlesRecord,
      renderPreview(container, preview) {
        for (const item of preview.questBonusRestores || []) {
          const section = node("section", undefined, "impact-group bonus-replacement-preview");
          section.append(node("h3", `${chapterTitle(item.chapterId)} · ${item.scheduleOnly ? "共鸣期限联动" : "还原共鸣"}`), node("p", `全部共鸣：${range(item.startDatetime, item.endDatetime)} · 期限组 ${item.termGroupId}`));
          section.append(node("p", `服装：${(item.costumeIds || []).map(id => title(draft.costumes.get(String(id))) || id).join(" / ") || "无"}`));
          const phases = item.groups || [];
          if (item.scheduleOnly) { section.append(node("p", `同步 ${phases.reduce((sum, group) => sum + group.questIds.length, 0)} 个关卡的成员期限，加成效果不变。`)); container.append(section); continue; }
          const families = new Map();
          for (const id of unique(phases.flatMap(group => (group.weapons || []).map(row => row.weaponId)))) {
            const tiers = phases.map(group => (group.weapons || []).filter(row => row.weaponId === id).sort((a, b) => a.limitBreak - b.limitBreak));
            const key = `${draft.family(id)}:${JSON.stringify(tiers.map(rows => rows.map(row => [row.limitBreak, row.rewards])))}`;
            if (!families.has(key)) families.set(key, { ids: [], tiers }); families.get(key).ids.push(id);
          }
          section.append(node("p", "额外数量依次为 0 / 1 / 2 / 3 / 4 突破；相同曲线的奖章合并显示。"));
          for (let start = 0; start < phases.length; start += 3) {
            const table = node("table", undefined, "bonus-preview-table"), header = node("tr"); header.append(node("th", "武器／形态"));
            for (const group of phases.slice(start, start + 3)) {
              const cell = node("th", `${group.questIds.length} 关`); cell.append(node("small", `${group.beforeBonusId} → ${group.afterBonusId}`), node("small", `参考 ${group.ruleBonusId}`)); cell.title = `关卡：${group.questIds.join(" / ")}`; header.append(cell);
            }
            const head = node("thead"); head.append(header); table.append(head); const body = node("tbody");
            for (const family of families.values()) {
              const row = node("tr"), name = node("td", title(draft.weapons.get(String(family.ids[0]))) || family.ids[0]); name.append(node("small", family.ids.join(" / "))); row.append(name);
              for (const tiers of family.tiers.slice(start, start + 3)) {
                const cell = node("td"), curves = new Map();
                for (const currency of unique(tiers.flatMap(tier => (tier.rewards || []).map(reward => reward.possessionId)))) {
                  const curve = [0, 1, 2, 3, 4].map(level => { const tier = tiers.filter(t => t.limitBreak <= level).at(-1); return (tier?.rewards || []).filter(r => r.possessionId === currency).reduce((sum, reward) => sum + reward.count, 0); }).join(" / ");
                  if (!curves.has(curve)) curves.set(curve, []); curves.get(curve).push(currency);
                }
                for (const [curve, ids] of curves) {
                  const rewards = node("div", ids.map(shortMedal).join("＋")); rewards.title = ids.map(id => `${medal(id)} · ${id}`).join(" / "); rewards.append(node("small", curve)); cell.append(rewards);
                }
                if (!curves.size) cell.append(node("span", "无掉落加成")); row.append(cell);
              }
              body.append(row);
            }
            table.append(body); section.append(table);
          }
          container.append(section);
        }
      }
    };
  };
})();
