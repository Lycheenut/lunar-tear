(() => {
  "use strict";
  const clone = value => JSON.parse(JSON.stringify(value));
  const key = member => `${member.kind}:${member.id}`;
  const localInput = value => {
    if (!value) return "";
    const date = new Date(value);
    return new Date(value - date.getTimezoneOffset() * 60000).toISOString().slice(0, 19);
  };
  const formatTime = value => value ? new Date(value).toLocaleString() : "未配置";

  window.createActivityGroupEditor = ({ root, api, showNotice, hasOtherChanges, onPublished }) => {
    let data = null, draft = null, baseline = "", mode = "groups", selected = "", search = "", busy = false;
    const dirty = () => draft && JSON.stringify(draft) !== baseline;
    const optionFor = member => data.catalog.options.find(option => key(option) === key(member));
    const title = member => {
      const option = optionFor(member);
      return `${option?.titles?.en || option?.titles?.ja || option?.titles?.ko || member.kind} · ${member.id}${option ? "" : "（引用已失效）"}`;
    };
    const kindLabel = kind => data.catalog.kinds.find(item => item.kind === kind)?.label || kind;
    const el = (tag, text, className) => {
      const node = document.createElement(tag);
      if (text != null) node.textContent = text;
      if (className) node.className = className;
      return node;
    };
    const button = (text, action, className = "button ghost") => {
      const node = el("button", text, className); node.type = "button"; node.disabled = busy;
      node.addEventListener("click", action); return node;
    };
    const field = (text, input) => { const label = el("label", text); label.append(input); return label; };
    const input = (value, onInput, type = "text") => {
      const node = el("input"); node.type = type; node.value = value; node.disabled = busy;
      node.addEventListener("input", () => onInput(node.value)); return node;
    };
    function select(options, value, onChange, label) {
      const node = el("select"); node.setAttribute("aria-label", label); node.disabled = busy;
      options.forEach(([id, name]) => node.append(new Option(name, id)));
      node.value = value; node.addEventListener("change", () => onChange(node.value)); return node;
    }
    async function run(action) {
      if (busy) return;
      busy = true; render();
      try { await action(); } catch (error) { showNotice(error.message, true); }
      finally { busy = false; render(); }
    }
    function envelope() { return { expectedContentHash: data.contentHash, expectedGachaConfigHash: data.gachaConfigHash, expectedMasterDataHash: data.masterDataHash }; }
    async function load(force = false) {
      if (data && !force) { render(); return; }
      data = await api("/api/admin/activity-groups");
      draft = clone(data.catalog.config); baseline = JSON.stringify(draft); render();
    }
    async function save() {
      if (hasOtherChanges()) throw new Error("请先保存或放弃其他页面的修改，再保存活动组。");
      await api("/api/admin/activity-groups", { method: "POST", body: JSON.stringify({ ...envelope(), config: draft }) });
      await load(true); await onPublished(); showNotice("活动组配置已保存。");
    }
    function add() {
      const id = `${mode === "groups" ? "group" : "unit"}-${crypto.randomUUID()}`;
      draft[mode].push(mode === "groups" ? { id, name: "新活动组", unitIds: [] } : { id, name: "新活动单位", type: 1, members: [] });
      selected = id; search = ""; render();
    }
    function remove(item) {
      if (mode === "units" && draft.groups.some(group => group.unitIds.includes(item.id))) { showNotice("该单位仍被活动组引用，请先从活动组中移除。", true); return; }
      if (!window.confirm(`删除“${item.name}”？保存配置后生效。`)) return;
      draft[mode] = draft[mode].filter(row => row.id !== item.id); selected = ""; render();
    }
    function updateSaveState() {
      const saveButton = root.querySelector("[data-group-save]");
      if (saveButton) saveButton.disabled = busy || !dirty();
      const summary = root.querySelector("[data-group-summary]");
      if (summary) summary.textContent = dirty() ? "有未保存的活动组修改" : "活动组配置已保存";
      root.querySelectorAll("[data-group-schedule]").forEach(node => { node.disabled = busy || dirty(); });
    }
    function render() {
      root.replaceChildren();
      if (!draft) return;
      const heading = el("div", null, "data-heading");
      const copy = el("div"); copy.append(el("p", "ACTIVITY GROUPS", "eyebrow"), el("h2", "活动组"));
      heading.append(copy, button("刷新", () => {
        if (dirty() && !window.confirm("刷新会放弃活动组中未保存的修改，是否继续？")) return;
        run(() => load(true));
      })); root.append(heading);
      const toolbar = el("div", null, "activity-group-toolbar");
      for (const [value, label] of [["groups", "活动组"], ["units", "活动单位"]]) {
        const tab = button(`${label} · ${draft[value].length}`, () => { mode = value; selected = ""; search = ""; render(); }, `button ${mode === value ? "primary" : "ghost"}`);
        tab.setAttribute("aria-pressed", String(mode === value)); toolbar.append(tab);
      }
      toolbar.append(button(mode === "groups" ? "新建活动组" : "新建活动单位", add)); root.append(toolbar);
      const layout = el("div", null, "activity-group-layout"), sidebar = el("aside", null, "activity-group-sidebar"), detail = el("section", null, "activity-group-detail");
      const searchInput = input(search, value => { search = value; renderList(); }, "search"); searchInput.placeholder = "搜索名称或 ID"; searchInput.setAttribute("aria-label", "搜索活动组或单位");
      const list = el("div", null, "activity-group-list"); sidebar.append(searchInput, list);
      function renderList() {
        list.replaceChildren();
        const rows = draft[mode].filter(item => `${item.name} ${item.id}`.toLowerCase().includes(search.toLowerCase()));
        if (!rows.length) list.append(el("p", "暂无匹配内容"));
        rows.forEach(item => {
          const row = button(item.name, () => { selected = item.id; render(); }, `activity-group-list-item${selected === item.id ? " active" : ""}`);
          row.title = item.id;
          row.append(el("small", mode === "groups" ? `${item.unitIds.length} 个单位` : `类型 ${item.type} · ${item.members.length} 个成员`)); list.append(row);
        });
      }
      renderList();
      const item = draft[mode].find(row => row.id === selected);
      if (!item) detail.append(el("p", "选择已有内容，或新建活动组 / 活动单位。", "activity-group-note"));
      else {
        const name = input(item.name, value => { item.name = value; renderList(); updateSaveState(); });
        name.maxLength = 200;
        const header = el("div", null, "activity-group-toolbar"); header.append(field("名称", name), button("删除", () => remove(item))); detail.append(header);
        if (mode === "units") renderUnit(detail, item); else renderGroup(detail, item);
      }
      layout.append(sidebar, detail); root.append(layout);
      const footer = el("div", null, "save-bar"); const summary = el("span"); summary.dataset.groupSummary = "";
      const actions = el("div", null, "save-actions");
      const saveButton = button("保存活动组配置", () => run(save), "button primary"); saveButton.dataset.groupSave = "";
      actions.append(button("放弃修改", () => { draft = clone(data.catalog.config); render(); }), saveButton); footer.append(summary, actions); root.append(footer); updateSaveState();
    }
    function renderUnit(detail, unit) {
      detail.append(field("单位类型", select([[1, "类型 1 · 活动副本"], [2, "类型 2 · Premium Gacha"]], unit.type, value => { unit.type = Number(value); render(); }, "单位类型")));
      detail.append(el("p", unit.type === 1 ? "至少添加 1 个 EventQuestChapter；Event Gacha 需要包含对应的 Variation 副本。" : "至少添加 1 个 Premium Gacha。", "activity-group-note"));
      const table = el("table", null, "activity-group-members");
      const head = el("thead"), header = el("tr"); ["类型", "成员", "当前时间", "操作"].forEach(text => header.append(el("th", text))); head.append(header); table.append(head);
      const body = el("tbody");
      unit.members.forEach((member, index) => {
        const row = el("tr"), option = optionFor(member), action = el("td"); action.append(button("移除", () => { unit.members.splice(index, 1); render(); }));
        row.append(el("td", kindLabel(member.kind)), el("td", title(member)), el("td", option ? `${formatTime(option.startDatetime)} → ${formatTime(option.endDatetime)}` : "引用已失效"), action); body.append(row);
      }); table.append(body); detail.append(table);
      const addRow = el("div", null, "activity-group-toolbar"); let memberKind = unit.type === 1 ? "chapter" : "premium", memberID = "";
      const memberHost = el("span", null, "activity-group-member-picker");
      function renderOptions() {
        const options = data.catalog.options.filter(option => option.kind === memberKind && !unit.members.some(member => key(member) === key(option)) && (memberKind !== "event" || unit.members.some(member => member.kind === "chapter" && member.id === option.relatedChapterId && optionFor(member)?.chapterType === 2)));
        memberID = "";
        memberHost.replaceChildren(select([["", "请选择成员"], ...options.map(option => [option.id, title(option)])], "", value => { memberID = value; }, "添加活动成员"));
      }
      addRow.append(select(data.catalog.kinds.filter(kind => kind.types.includes(unit.type)).map(kind => [kind.kind, kind.label]), memberKind, value => { memberKind = value; renderOptions(); }, "成员类型"), memberHost, button("添加成员", () => {
        if (!memberID) return; unit.members.push({ kind: memberKind, id: Number(memberID) }); render();
      })); renderOptions(); detail.append(addRow);
    }
    function renderGroup(detail, group) {
      detail.append(el("h3", "活动单位"));
      const unitList = el("div", null, "activity-group-unit-list");
      group.unitIds.forEach(id => {
        const unit = draft.units.find(item => item.id === id), row = el("div", null, "activity-group-toolbar");
        row.append(el("span", unit ? `${unit.name} · 类型 ${unit.type}` : `${id}（引用已失效）`), button("编辑单位", () => { mode = "units"; selected = id; search = ""; render(); }), button("移除", () => { group.unitIds = group.unitIds.filter(value => value !== id); render(); })); unitList.append(row);
      }); detail.append(unitList);
      let unitID = ""; const addRow = el("div", null, "activity-group-toolbar");
      addRow.append(select([["", "选择活动单位"], ...draft.units.filter(unit => !group.unitIds.includes(unit.id)).map(unit => [unit.id, unit.name])], "", value => { unitID = value; }, "组合活动单位"), button("加入活动组", () => { if (unitID) { group.unitIds.push(unitID); render(); } })); detail.append(addRow);
      detail.append(el("h3", "整体修改时间"), el("p", "使用本机时区。副本、Premium Gacha、Banner、NaviCutIn 和 Tip 使用活动起止时间；活动任务、Event Gacha、兑换商店与道具 / 碎片有效期延长至结束后 48 小时。请先保存配置，再预览改时。", "activity-group-note"));
      const source = draft.units.filter(unit => group.unitIds.includes(unit.id)).flatMap(unit => unit.members).find(member => member.kind === "chapter" || member.kind === "premium");
      const option = source && optionFor(source);
      const start = input(localInput(option?.startDatetime), () => {}, "datetime-local"), end = input(localInput(option?.endDatetime), () => {}, "datetime-local");
      start.step = "1"; end.step = "1";
      const controls = el("div", null, "activity-group-toolbar");
      const previewButton = button("预览整体改时", () => {
        const startTime = new Date(start.value).getTime(), endTime = new Date(end.value).getTime();
        if (!Number.isFinite(startTime) || !Number.isFinite(endTime) || startTime >= endTime) { showNotice("请输入有效的起止时间，结束时间必须晚于开始时间。", true); return; }
        if (hasOtherChanges()) { showNotice("请先保存或放弃其他页面的修改，再修改活动组时间。", true); return; }
        const request = { ...envelope(), groupId: group.id, startDatetime: startTime, endDatetime: endTime };
        run(async () => {
          const preview = await api("/api/admin/activity-groups/schedule/preview", { method: "POST", body: JSON.stringify(request) });
          showPreview(group, request, preview.changes);
        });
      }, "button primary"); previewButton.dataset.groupSchedule = "";
      controls.append(field("活动开始", start), field("活动结束", end), previewButton); detail.append(controls);
    }
    function showPreview(group, request, changes) {
      const dialog = el("dialog", null, "confirm-dialog activity-group-dialog");
      dialog.append(el("h2", `${group.name} · 改时预览`), el("p", `${changes.length} 个字段将被修改。共享成员只更新一次，也会影响使用这些成员的其他活动组。`));
      const scroll = el("div", null, "activity-group-preview-scroll"), table = el("table");
      const header = el("tr"); ["成员", "字段", "修改前", "修改后"].forEach(text => header.append(el("th", text))); table.append(header);
      changes.forEach(change => { const row = el("tr"); [title(change), change.field, formatTime(change.before), formatTime(change.after)].forEach(text => row.append(el("td", text))); table.append(row); });
      scroll.append(table); dialog.append(scroll);
      const errorMessage = el("p", "", "notice error hidden"); dialog.append(errorMessage);
      const actions = el("div", null, "save-actions");
      const close = button("取消", () => dialog.close()); close.disabled = false;
      const submit = button("确认整体改时", async () => {
        submit.disabled = true; close.disabled = true;
        errorMessage.classList.add("hidden");
        try {
          await api("/api/admin/activity-groups/schedule", { method: "POST", body: JSON.stringify(request) });
          dialog.close(); await load(true); await onPublished(); showNotice("活动组时间已更新，兑换期限保留 48 小时。");
        } catch (error) { errorMessage.textContent = error.message; errorMessage.classList.remove("hidden"); }
        finally { submit.disabled = false; close.disabled = false; }
      }, "button primary"); submit.disabled = !changes.length;
      actions.append(close, submit); dialog.append(actions); document.body.append(dialog);
      dialog.addEventListener("close", () => dialog.remove()); dialog.showModal();
    }
    return { load, dirty, reset: () => { data = null; draft = null; baseline = ""; } };
  };
})();
