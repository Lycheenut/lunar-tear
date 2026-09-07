(() => {
  "use strict";
  const bonusTable = "m_quest_bonus", costumeTable = "m_quest_bonus_costume_setting_group", weaponTable = "m_quest_bonus_weapon_group";
  const effectTable = "m_quest_bonus_effect_group", dropTable = "m_quest_bonus_drop_reward";
  const text = titles => Object.values(titles || {}).join(" ");
  const unique = values => [...new Set(values)];

  class QuestBonusDraft {
    constructor(catalog) {
      this.catalog = catalog || { tables: [], chapters: [], quests: [], costumes: [], weapons: [], medals: [] };
      this.groups = new Map(); this.bonuses = new Map(); this.selections = new Map(); this.modes = new Map(); this.membersCache = new Map(); this.profilesCache = new Map();
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
        const medals = [...(q.medalIds || [])].sort((a, b) => a - b).join(",");
        const bonus = this.bonuses.get(String(q.bonusId));
        const signature = JSON.stringify([bonus ? Object.entries(bonus).filter(([name]) => name !== "QuestBonusId").sort() : q.bonusId, medals]);
        if (!groups.has(signature)) groups.set(signature, { key: `${q.bonusId}:${medals}`, bonusID: String(q.bonusId), quests: [], medalIDs: q.medalIds || [] });
        groups.get(signature).quests.push(q);
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
        const signature = JSON.stringify(phases);
        if (!merged.has(signature)) merged.set(signature, { id: member.itemID, members: [], complete: phases.every(rows => rows.length > 0) });
        merged.get(signature).members.push(member);
      }
      const result = [...merged.values()]; this.profilesCache.set(key, result); return result;
    }
    templateRows(bonusID, weaponID, templateID) {
      const rows = this.weaponRows(bonusID).filter(row => this.family(row.WeaponId) === this.family(templateID) && this.order(row.WeaponId) <= this.order(weaponID));
      const order = Math.max(-1, ...rows.map(row => this.order(row.WeaponId)));
      return rows.filter(row => this.order(row.WeaponId) === order);
    }
    ruleID(chapterID, selected, group) { return selected.ruleChapterID === String(chapterID) ? group.bonusID : selected.groups[group.key]; }
    template(selected, member, group) { return selected.phaseChoices[member.key]?.[group.key] ?? selected.choices[member.key]; }
    supports(bonusID, member, templateID) { return !!templateID && member.itemIDs.every(id => this.templateRows(bonusID, id, templateID).length > 0); }
    setChoice(selected, member, value) { selected.choices[member.key] = value; delete selected.phaseChoices[member.key]; }
    requiredMedals(chapterID, selected) {
      return unique(this.selectedMembers(chapterID).filter(m => m.kind === "武器").flatMap(member => this.targetGroups(chapterID).flatMap(group =>
        member.itemIDs.flatMap(id => this.templateRows(this.ruleID(chapterID, selected, group), id, this.template(selected, member, group)).flatMap(row => this.rewards(row.QuestBonusEffectGroupId).map(reward => reward.possessionId))))));
    }
    mode(chapterID) { return this.modes.get(String(chapterID)) || "replace"; }
    setMode(chapterID, mode) {
      if (this.mode(chapterID) === mode) return;
      this.modes.set(String(chapterID), mode);
      const selected = this.selections.get(String(chapterID));
      if (selected) this.replace(chapterID, selected.sourceBonusID);
    }
    included(chapterID, member) { return this.mode(chapterID) === "append" && this.members(this.currentIDs(chapterID)).has(member.key); }
    selectedMembers(chapterID) {
      const selected = this.selections.get(String(chapterID));
      return selected ? [...this.members([selected.sourceBonusID]).values()].filter(member => selected.members.has(member.key) && !this.included(chapterID, member)) : [];
    }
    setMember(chapterID, member, checked) {
      const selected = this.selections.get(String(chapterID));
      if (!selected || this.included(chapterID, member)) return;
      if (checked) selected.members.add(member.key); else selected.members.delete(member.key);
    }
    changed(chapterID) { return this.selections.has(String(chapterID)) && (this.mode(chapterID) === "replace" || this.selectedMembers(chapterID).length > 0); }
    summary(chapterID) {
      const current = this.members(this.currentIDs(chapterID)), chosen = this.selectedMembers(chapterID);
      const final = new Map(this.mode(chapterID) === "append" ? current : []);
      for (const member of chosen) final.set(member.key, member);
      const count = members => { const list = [...members]; return ["服装", "武器"].map(kind => list.filter(m => m.kind === kind).length); };
      return { final: count(final.values()), added: count(chosen.filter(m => !current.has(m.key))), removed: count([...current.values()].filter(m => !final.has(m.key))) };
    }
    replace(chapterID, bonusID) {
      const key = String(chapterID);
      if (bonusID === "") { this.selections.delete(key); return; }
      if (!this.quests(key).length) throw new Error("该活动没有可还原的关卡。");
      if (!this.bonuses.has(String(bonusID))) throw new Error("请选择一个已有加成条目。");
      const ids = new Set(this.quests(key).map(q => q.questId));
      if (this.catalog.quests.some(q => String(q.chapterId) !== key && ids.has(q.questId))) throw new Error("此活动与其他活动共用关卡，需先分离关卡引用才能使用独立活动期限。");
      const selected = { sourceBonusID: String(bonusID), ruleChapterID: this.selections.get(key)?.ruleChapterID || key, members: new Set(this.mode(key) === "replace" ? this.members([bonusID]).keys() : []), choices: {}, groups: {}, currencies: {} };
      this.selections.set(key, selected); this.setReference(key, selected.ruleChapterID);
    }
    setReference(chapterID, ruleChapterID) {
      const selected = this.selections.get(String(chapterID));
      selected.ruleChapterID = String(ruleChapterID); selected.choices = {}; selected.phaseChoices = {}; selected.groups = {}; selected.currencies = {};
      const profiles = this.profiles(ruleChapterID);
      for (const member of this.members([selected.sourceBonusID]).values()) if (member.kind === "武器") {
        const profile = profiles.find(p => p.members.some(m => m.key === member.key));
        if (profile) selected.choices[member.key] = profile.id;
      }
    }
    count() { return [...this.selections.keys()].filter(id => this.changed(id)).length; }
    payload() {
      const restores = [];
      for (const [chapterID, selected] of this.selections) {
        if (!this.changed(chapterID)) continue;
        const members = this.selectedMembers(chapterID), source = members.filter(m => m.kind === "武器");
        const external = selected.ruleChapterID !== chapterID;
        const groups = this.targetGroups(chapterID);
        if (source.length && !external && groups.some(group => group.bonusID === "0")) throw new Error(`活动 ${chapterID} 缺少现有加成，请选择规则参考活动并设置对应关系。`);
        if (source.length && external && groups.some(group => !selected.groups[group.key])) throw new Error(`活动 ${chapterID} 尚未完成关卡分组对应。`);
        const weaponPhases = groups.map(group => ({ questIds: group.quests.map(q => q.questId), weapons: [] }));
        const weapons = source.map(member => {
          const template = selected.choices[member.key] || groups.map(group => this.template(selected, member, group)).find(Boolean);
          if (!template) throw new Error(`活动 ${chapterID}：${text(member.titles) || member.itemID} 尚未选择武器规则。`);
          groups.forEach((group, index) => {
            const choice = this.template(selected, member, group);
            if (!this.profiles(selected.ruleChapterID).some(p => p.id === choice) || !this.supports(this.ruleID(chapterID, selected, group), member, choice)) throw new Error(`活动 ${chapterID}：${text(member.titles) || member.itemID} 在关卡 ${group.quests[0].questId} 等 ${group.quests.length} 关缺少适用规则，请通过「分阶段」补全。`);
            if (choice !== template) weaponPhases[index].weapons.push({ weaponId: Number(member.itemID), templateWeaponId: Number(choice) });
          });
          return { weaponId: Number(member.itemID), templateWeaponId: Number(template) };
        });
        const input = { chapterId: Number(chapterID), sourceBonusId: Number(selected.sourceBonusID), mode: this.mode(chapterID), costumeIds: members.filter(m => m.kind === "服装").map(m => Number(m.itemID)), ruleChapterId: Number(selected.ruleChapterID), weapons };
        if (weaponPhases.some(group => group.weapons.length)) input.weaponPhases = weaponPhases.filter(group => group.weapons.length);
        if (source.length && external) {
          if (this.requiredMedals(chapterID, selected).some(id => !selected.currencies[id])) throw new Error(`活动 ${chapterID} 尚未完成奖章对应。`);
          input.groups = groups.map(group => ({ questIds: group.quests.map(q => q.questId), ruleBonusId: Number(selected.groups[group.key]) }));
          input.currencies = Object.entries(selected.currencies).filter(([, value]) => value).map(([from, to]) => ({ fromId: Number(from), toId: Number(to) }));
        }
        restores.push(input);
      }
      return { changes: [], questBonusGroups: [], questBonusRestores: restores };
    }
  }
  window.QuestBonusDraft = QuestBonusDraft;

  window.createQuestBonusEditor = ({ root, searchRoot, onChange, localizedText, showError, formatDatetime }) => {
    let draft = new QuestBonusDraft(), chapterID = "", phaseIndex = 0;
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
    const select = (label, entries, value, action) => {
      const el = node("select"); el.setAttribute("aria-label", label); el.dataset.searchable = "true";
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
      const rules = draft.targetGroups(selected.ruleChapterID).filter(group => group.bonusID !== "0").map(group => ({ value: group.bonusID, label: `${group.bonusID} · ${groupLabel(group)}` }));
      const grid = node("div", undefined, "bonus-mapping-grid");
      for (const group of draft.targetGroups(chapterID)) grid.append(field(groupLabel(group), select(`参考分组 ${group.key}`, [{ value: "", label: "选择参考关卡分组…" }, ...rules], selected.groups[group.key] || "", value => { selected.groups[group.key] = value; renderCurrencies(); window.AdminSearchSelect?.refresh(); onChange(); })));
      dialog.append(node("h3", "目标关卡 → 参考分组"), grid, node("h3", "参考奖章 → 目标奖章"));
      const currencies = node("div", undefined, "bonus-mapping-grid");
      const available = unique(draft.quests(chapterID).flatMap(q => q.medalIds || []));
      function renderCurrencies() {
        currencies.replaceChildren();
        const required = draft.requiredMedals(chapterID, selected);
        for (const id of required) currencies.append(field(`${medal(id)} · ${id}`, select(`目标奖章 ${id}`, [{ value: "", label: "选择本活动实际掉落的奖章…" }, ...available.map(value => ({ value: String(value), label: `${medal(value)} · ${value}` }))], selected.currencies[id] || "", value => { selected.currencies[id] = value; onChange(); })));
        if (!required.length) currencies.append(node("p", "先选择武器规则，再设置这些规则使用的奖章。", "bonus-note"));
      }
      renderCurrencies();
      dialog.append(currencies, button("完成", () => dialog.close()));
      dialog.addEventListener("close", () => { dialog.remove(); render(); }); document.body.append(dialog); window.AdminSearchSelect?.refresh(); dialog.showModal();
    }
    function rewardsAt(bonusID, weaponID, templateID, selected) {
      const rows = draft.templateRows(bonusID, weaponID, templateID).sort((a, b) => Number(a.LimitBreakCountLowerLimit) - Number(b.LimitBreakCountLowerLimit));
      if (!rows.length) return "本阶段未配置加成";
      const tiers = [0, 1, 2, 3, 4].map(level => {
        const row = rows.filter(row => Number(row.LimitBreakCountLowerLimit) <= level).at(-1);
        return row ? draft.rewards(row.QuestBonusEffectGroupId).map(reward => ({ ...reward, possessionId: Number(selected?.currencies[reward.possessionId] || reward.possessionId) })) : [];
      });
      const curves = new Map();
      for (const id of unique(tiers.flat().map(reward => reward.possessionId))) {
        const curve = tiers.map(rewards => `+${rewards.filter(reward => reward.possessionId === id).reduce((sum, reward) => sum + reward.count, 0)}`).join(" / ");
        if (!curves.has(curve)) curves.set(curve, []); curves.get(curve).push(id);
      }
      return [...curves].map(([curve, ids]) => `${ids.map(shortMedal).join("＋")} ${curve}`).join("；") || "无掉落加成";
    }
    function phaseDialog(selected, member) {
      const dialog = node("dialog", undefined, "bonus-mapping-dialog bonus-phase-dialog");
      dialog.setAttribute("aria-label", `分阶段规则 ${member.itemID}`);
      dialog.append(node("h2", `${title(member)} · 分阶段规则`), node("p", "各阶段可选择不同武器作为规则参考；数量依次为 0 / 1 / 2 / 3 / 4 突破。", "bonus-note"));
      const table = node("table", undefined, "bonus-preview-table"), head = node("tr");
      head.append(node("th", "关卡阶段"), node("th", "参考武器／加成")); table.append(head);
      for (const group of draft.targetGroups(chapterID)) {
        const ruleID = draft.ruleID(chapterID, selected, group), row = node("tr"), cell = node("td");
        const profiles = draft.profiles(selected.ruleChapterID).filter(profile => draft.supports(ruleID, member, profile.id));
        const value = draft.template(selected, member, group), info = node("small");
        const updateInfo = choice => {
          const ready = draft.supports(ruleID, member, choice);
          info.textContent = ready ? unique(member.itemIDs.map(id => rewardsAt(ruleID, id, choice, selected))).join("；") : ruleID ? "待选择本阶段规则" : "请先设置关卡分组对应";
          info.className = ready ? "" : "bonus-warning";
        };
        const picker = select(`阶段规则 ${group.key}`, [{ value: "", label: ruleID ? "选择本阶段规则…" : "先设置关卡分组对应…" }, ...profiles.map(profile => ({ value: profile.id, label: profile.members.map(title).join(" / "), search: profile.members.flatMap(m => [...m.itemIDs, text(m.titles)]).join(" ") }))], profiles.some(profile => profile.id === value) ? value : "", choice => {
          (selected.phaseChoices[member.key] ||= {})[group.key] = choice; updateInfo(choice); onChange();
        });
        picker.disabled = !ruleID; updateInfo(value);
        const caption = node("td", groupLabel(group)); caption.title = `${groupLabel(group)}；关卡 ${group.quests.map(q => q.questId).join(" / ")}`;
        cell.append(picker, info); row.append(caption, cell); table.append(row);
      }
      dialog.append(table, button("完成", () => dialog.close()));
      dialog.addEventListener("close", () => { dialog.remove(); render(); }); document.body.append(dialog); window.AdminSearchSelect?.refresh(); dialog.showModal();
    }
    function roster(ids, label, selected, phase) {
      const panel = node("section", undefined, "bonus-roster");
      const members = [...draft.members(ids).values()], costumes = members.filter(m => m.kind === "服装"), weapons = members.filter(m => m.kind === "武器");
      panel.append(node("h3", `${label} · ${costumes.length} 套服装 / ${weapons.length} 种武器`));
      if (selected) {
        const actions = node("div", undefined, "bonus-roster-actions");
        for (const kind of ["服装", "武器"]) {
          actions.append(node("span", kind));
          for (const [caption, checked] of [["全选", true], ["清空", false]]) {
            const action = button(caption, () => { for (const member of members.filter(m => m.kind === kind)) draft.setMember(chapterID, member, checked); onChange(); render(); }, "bonus-text-button");
            action.setAttribute("aria-label", `${kind}${caption}`); actions.append(action);
          }
        }
        panel.append(actions);
      }
      const memberLabel = member => {
        const included = draft.included(chapterID, member), checked = included || selected.members.has(member.key);
        const label = node("label", undefined, `bonus-member${included ? " bonus-included" : ""}`), checkbox = node("input");
        checkbox.type = "checkbox"; checkbox.checked = checked; checkbox.disabled = included;
        checkbox.setAttribute("aria-label", `${member.kind} ${title(member)} · ${member.itemID}`);
        checkbox.addEventListener("change", () => { draft.setMember(chapterID, member, checkbox.checked); onChange(); render(); });
        label.title = `${title(member)} · ${member.itemIDs.join(" / ")}`;
        label.append(checkbox, node("span", title(member)));
        if (included) label.append(node("small", "已包含"));
        return label;
      };
      const names = node("div", undefined, "bonus-costume-list");
      for (const member of costumes) {
        if (selected) names.append(memberLabel(member));
        else { const chip = node("span", title(member)); chip.title = `${title(member)} · ${member.itemIDs.join(" / ")}`; names.append(chip); }
      }
      if (!costumes.length) names.append(node("span", "无共鸣服装", "bonus-note"));
      panel.append(names);
      if (!weapons.length) panel.append(node("p", "无共鸣武器", "bonus-note"));
      const profiles = selected ? draft.profiles(selected.ruleChapterID) : [];
      for (const member of weapons) {
        const row = node("div", undefined, "bonus-weapon-row");
        if (selected) {
          row.append(memberLabel(member));
          if (draft.included(chapterID, member) || !selected.members.has(member.key)) {
            row.classList.add("bonus-weapon-unselected"); panel.append(row); continue;
          }
        } else {
          const name = node("strong", title(member)); name.title = `${title(member)} · ${member.itemIDs.join(" / ")}`; row.append(name);
        }
        if (selected) {
          const composed = Object.keys(selected.phaseChoices[member.key] || {}).length > 0;
          const picker = select(`武器规则 ${member.itemID}`, [{ value: "", label: "选择武器规则…" }, ...(composed ? [{ value: "已按阶段组合", label: "已按阶段组合" }] : []), ...profiles.map((profile, index) => ({ value: profile.id, label: `规则 ${index + 1} · ${profile.members.map(title).join(" / ")}${profile.complete ? "" : " · 部分阶段"}`, search: profile.members.flatMap(m => [...m.itemIDs, text(m.titles)]).join(" ") }))], composed ? "已按阶段组合" : selected.choices[member.key] || "", value => { if (value === "已按阶段组合") return; draft.setChoice(selected, member, value); onChange(); render(); });
          const controls = node("div", undefined, "bonus-rule-controls");
          const missing = draft.targetGroups(chapterID).filter(group => !draft.supports(draft.ruleID(chapterID, selected, group), member, draft.template(selected, member, group))).length;
          const action = button(missing ? `分阶段 · 缺 ${missing}` : "分阶段", () => phaseDialog(selected, member), "bonus-text-button");
          action.setAttribute("aria-label", `分阶段规则 ${member.itemID}`); controls.append(picker, action); row.append(controls);
          const template = phase && draft.template(selected, member, phase), ruleID = phase && draft.ruleID(chapterID, selected, phase);
          const ready = draft.supports(ruleID, member, template);
          const values = ready ? unique(member.itemIDs.map(id => rewardsAt(ruleID, id, template, selected))) : [ruleID ? "本阶段缺少规则，请通过「分阶段」选择" : "待选择关卡分组对应"];
          const info = node("small", values.join("；"), ready ? "" : "bonus-warning"); info.title = values.join("；"); row.append(info);
        } else {
          const values = phase ? unique(member.itemIDs.map(id => rewardsAt(phase.bonusID, id, id))) : [];
          const info = node("small", values.join("；")); info.title = `${values.join("；")}；当前名单汇总全活动成员，数量按所选关卡阶段显示。`; row.append(info);
        }
        panel.append(row);
      }
      return panel;
    }
    function render() {
      const scrollContainer = root.closest(".table-scroll"), scrollTop = scrollContainer?.scrollTop || 0;
      const focused = document.activeElement, focusLabel = root.contains(focused) && focused.type === "checkbox" ? focused.getAttribute("aria-label") : null;
      root.replaceChildren(); searchRoot.replaceChildren(); if (!draft.catalog.chapters.length) return;
      searchRoot.append(select("目标活动", [{ value: "", label: "搜索活动标题或 ID…" }, ...draft.catalog.chapters.map(row => ({ value: row.values.EventQuestChapterId, label: `${chapterTitle(row.values.EventQuestChapterId)} · ${row.values.EventQuestChapterId}`, search: text(row.titles) }))], chapterID, value => { chapterID = value; phaseIndex = 0; render(); if (scrollContainer) scrollContainer.scrollTop = 0; }));
      const main = node("section", undefined, "bonus-replacement"); root.append(main);
      if (!chapter(chapterID)) { main.append(node("p", "请选择目标活动。", "bonus-note")); window.AdminSearchSelect?.refresh(); return; }
      const currentIDs = draft.currentIDs(chapterID), selected = draft.selections.get(chapterID), groups = draft.targetGroups(chapterID);
      phaseIndex = Math.min(phaseIndex, Math.max(0, groups.length - 1)); const phase = groups[phaseIndex];
      const heading = node("div", undefined, "bonus-replacement-heading"), modes = node("div", undefined, "bonus-modes");
      modes.setAttribute("role", "group"); modes.setAttribute("aria-label", "名单编辑模式");
      for (const [value, caption] of [["replace", "替换名单"], ["append", "保留当前并补充"]]) {
        const action = button(caption, () => { draft.setMode(chapterID, value); onChange(); render(); }, "bonus-mode");
        action.setAttribute("aria-pressed", String(draft.mode(chapterID) === value)); modes.append(action);
      }
      const activity = node("div");
      activity.append(node("h2", chapterTitle(chapterID)), node("p", range(chapter(chapterID).values.StartDatetime, chapter(chapterID).values.EndDatetime), "bonus-note bonus-activity-dates"));
      heading.append(activity, modes, node("span", `${chapterID} · ${draft.quests(chapterID).length} 关卡`, "row-badge")); main.append(heading);
      const pickers = node("div", undefined, "bonus-source-picker");
      const sources = [...draft.bonuses].sort(([a], [b]) => Number(a) - Number(b)).map(([id, bonus]) => {
        const members = [...draft.members([id]).values()];
        const representative = kind => { const entries = members.filter(m => m.kind === kind); return entries.length ? `${title(entries[0])}${entries.length > 1 ? " 等" : ""}` : "无"; };
        return { value: id, label: `${id} · 服装组 ${bonus.QuestBonusCostumeSettingGroupId} (${representative("服装")}) / 武器组 ${bonus.QuestBonusWeaponGroupId} (${representative("武器")})`, search: [...Object.values(bonus), ...members.flatMap(m => [text(m.titles), ...m.itemIDs])].join(" "), group: Object.entries(bonus).filter(([name]) => name !== "QuestBonusId").map(([, value]) => value).join(":") };
      });
      pickers.append(field("历史名单来源", select("历史名单来源", [{ value: "", label: "搜索加成 ID、组 ID 或服装／武器名称…" }, ...sources], selected?.sourceBonusID || "", value => { draft.replace(chapterID, value); onChange(); render(); })));
      const reference = select("规则参考活动", draft.catalog.chapters.filter(row => row.values.EventQuestChapterId === chapterID || draft.currentIDs(row.values.EventQuestChapterId).some(id => id !== "0")).map(row => ({ value: row.values.EventQuestChapterId, label: `${row.values.EventQuestChapterId === chapterID ? "本活动 · " : ""}${chapterTitle(row.values.EventQuestChapterId)} · ${row.values.EventQuestChapterId}` })), selected?.ruleChapterID || chapterID, value => { draft.setReference(chapterID, value); onChange(); render(); });
      reference.disabled = !selected; pickers.append(field(draft.mode(chapterID) === "append" ? "新增武器规则参考" : "武器规则参考", reference)); main.append(pickers);
      const controls = node("div", undefined, "bonus-phase-controls");
      controls.append(field("关卡预览", select("预览关卡分组", groups.map((group, index) => ({ value: String(index), label: groupLabel(group) })), String(phaseIndex), value => { phaseIndex = Number(value); render(); })), node("span", "突破 0 / 1 / 2 / 3 / 4", "bonus-note"));
      if (selected && selected.ruleChapterID !== chapterID && draft.selectedMembers(chapterID).some(m => m.kind === "武器")) controls.append(button("关卡／奖章对应", () => mappingDialog(selected)));
      main.append(controls);
      const comparison = node("div", undefined, "bonus-comparison");
      comparison.append(roster(currentIDs, draft.mode(chapterID) === "append" ? "当前名单（全部保留）" : "当前名单", null, phase));
      if (selected) comparison.append(roster([selected.sourceBonusID], "来源名单", selected, phase));
      else comparison.append(node("p", "请选择历史名单来源。", "bonus-note"));
      main.append(comparison);
      const note = node("div", undefined, "bonus-replacement-note");
      if (selected) {
        const summary = draft.summary(chapterID), counts = values => `${values[0]} 套服装 / ${values[1]} 种武器`;
        const result = node("p", `最终 ${counts(summary.final)} · 新增 ${counts(summary.added)} · 移除 ${counts(summary.removed)}`, "bonus-summary"); result.setAttribute("role", "status"); note.append(result);
        try { draft.payload(); }
        catch (error) { note.append(node("p", error.message, "bonus-warning")); }
        note.append(button("撤销本活动", () => { draft.replace(chapterID, ""); onChange(); render(); }));
      }
      main.append(note);
      window.AdminSearchSelect?.refresh(); if (scrollContainer) scrollContainer.scrollTop = scrollTop;
      if (focusLabel) [...root.querySelectorAll('input[type="checkbox"]')].find(input => input.getAttribute("aria-label") === focusLabel)?.focus({ preventScroll: true });
    }
    const handlesRecord = record => record.table === "m_quest" && record.changes?.some(change => change.field === "QuestBonusId");
    return {
      load(catalog) { draft = new QuestBonusDraft(catalog); }, reset() { draft.selections.clear(); draft.modes.clear(); },
      selectChapter(id) { chapterID = String(id); phaseIndex = 0; }, count: () => draft.count(), payload: () => draft.payload(), render, handlesRecord,
      renderPreview(container, preview) {
        for (const item of preview.questBonusRestores || []) {
          const section = node("section", undefined, "impact-group bonus-replacement-preview");
          section.append(node("h3", `${chapterTitle(item.chapterId)} · ${item.scheduleOnly ? "共鸣期限联动" : item.mode === "append" ? "补充共鸣" : "替换共鸣"}`), node("p", `全部共鸣：${range(item.startDatetime, item.endDatetime)} · 期限组 ${item.termGroupId}`));
          section.append(node("p", `服装：${(item.costumeIds || []).map(id => title(draft.costumes.get(String(id))) || id).join(" / ") || "无"}`));
          if (item.scheduleOnly) { section.append(node("p", `同步 ${(item.groups || []).reduce((sum, group) => sum + group.questIds.length, 0)} 个关卡的成员期限，加成效果不变。`)); container.append(section); continue; }
          const merged = new Map(), quests = new Map(draft.quests(item.chapterId).map(q => [q.questId, q]));
          for (const group of item.groups || []) {
            const signature = JSON.stringify([group.afterBonusId, [...(quests.get(group.questIds[0])?.medalIds || [])].sort((a, b) => a - b), group.weapons]);
            if (!merged.has(signature)) merged.set(signature, { ...group, questIds: [], beforeIDs: [], ruleIDs: [] });
            const phase = merged.get(signature); phase.questIds.push(...group.questIds); phase.beforeIDs.push(group.beforeBonusId); phase.ruleIDs.push(group.ruleBonusId);
          }
          const phases = [...merged.values()];
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
              const compact = ids => `${ids[0]}${unique(ids).length > 1 ? " 等" : ""}`;
              const cell = node("th", `${group.questIds.length} 关`); cell.append(node("small", `${compact(group.beforeIDs)} → ${group.afterBonusId}`)); if (group.ruleBonusId) cell.append(node("small", `参考 ${compact(group.ruleIDs)}`)); cell.title = `关卡：${group.questIds.join(" / ")}；原加成：${unique(group.beforeIDs).join(" / ")}；参考：${unique(group.ruleIDs).join(" / ")}`; header.append(cell);
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
