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
  const unitTypes = [[1, "Premium Gacha"], [2, "記録（Record）"], [3, "変異（Variation）"]];
  const sectionsByType = {
    1: [["Premium Gacha", ["premium"]], ["MomBanner", ["banner"]], ["兑换处", ["shop"]], ["碎片有效期", ["term"]]],
    2: [["記録（Record）", ["chapter"]], ["MomBanner", ["banner"]], ["活动商店", ["shop"]], ["兑换币有效期", ["term"]], ["活动任务", ["mission"]], ["NaviCutIn", ["navi"]], ["Tip", ["tip"]]],
    3: [["変異（Variation）", ["chapter"]], ["MomBanner", ["banner"]], ["Event Gacha", ["event"]], ["Event Gacha Ticket 有效期", ["term"]], ["活动任务", ["mission"]], ["NaviCutIn", ["navi"]], ["Tip", ["tip"]]]
  };

  window.createActivityGroupEditor = ({ root, api, showNotice, localizedText, renderBannerPreview, hasOtherChanges, onPublished }) => {
    let data = null, draft = null, baseline = "", mode = "groups", selected = "", search = "", busy = false;
    let optionIndex = new Map();
    const sidebarScroll = { groups: 0, units: 0 };
    let unitTypeFilter = "";
    const checkedIDs = { groups: new Set(), units: new Set() };
    const clearChecked = () => { checkedIDs.groups.clear(); checkedIDs.units.clear(); };
    const dirty = () => draft && JSON.stringify(draft) !== baseline;
    const optionFor = member => optionIndex.get(key(member));
    const title = member => {
      const option = optionFor(member);
      return `${member.id}. ${localizedText(option?.titles) || member.kind}${option ? "" : "（引用已失效）"}`;
    };
    const typeLabel = type => unitTypes.find(([value]) => value === type)?.[1] || "未知类型";
    const itemName = item => {
      const unit = item.members ? item : draft.units.find(unit => unit.id === item.id && item.unitIds?.includes(unit.id));
      const source = unit?.members.find(member => key(member) === item.id);
      const titles = source && optionFor(source)?.titles;
      return titles && Object.values(titles).includes(item.name) ? localizedText(titles) || item.name : item.name;
    };
    const itemTitle = item => item.members ? `${item.id.replace(/^(chapter|premium):/, "")}. ${itemName(item)}` : itemName(item);
    const visibleMembers = unit => [...new Map(unit.members.map(member => {
      const visible = unit.type === 1 && member.kind === "medal" ? { kind: "term", id: member.id } : member;
      return [key(visible), visible];
    })).values()];
    const memberAllowed = (option, unit) => {
      if (!option || !data.catalog.kinds.find(kind => kind.kind === option.kind)?.types.includes(unit.type)) return false;
      if (option.kind === "chapter") return option.chapterType === unit.type - 1;
      return option.kind !== "event" || unit.members.some(member => member.kind === "chapter" && member.id === option.relatedChapterId && optionFor(member)?.chapterType === 2);
    };
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
      options.forEach(([id, name]) => {
        const option = new Option(name, id); option.dataset.searchLabel = name; option.dataset.search = id; node.append(option);
      });
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
      optionIndex = new Map(data.catalog.options.map(option => [key(option), option]));
      draft = clone(data.catalog.config); baseline = JSON.stringify(draft); clearChecked(); render();
    }
    async function save(request) {
      if (hasOtherChanges()) throw new Error("请先保存或放弃其他页面的修改，再保存活动组。");
      await api("/api/admin/activity-groups", { method: "POST", body: JSON.stringify(request) });
      await load(true); await onPublished(); showNotice("活动组配置已保存。");
    }
    function configChanges(before, after) {
      const changes = [];
      const memberName = (member, type) => {
        const label = sectionsByType[type]?.find(([, kinds]) => kinds.includes(member.kind))?.[0]
          || data.catalog.kinds.find(kind => kind.kind === member.kind)?.label || member.kind;
        return `${label}：${title(member)}`;
      };
      const unitName = (id, config) => { const unit = config.units.find(unit => unit.id === id); return unit ? itemTitle(unit) : id; };
      for (const [collection, label] of [["groups", "活动组"], ["units", "活动单位"]]) {
        const oldItems = new Map(before[collection].map(item => [item.id, item])), newItems = new Map(after[collection].map(item => [item.id, item]));
        const entries = (item, config) => collection === "units" ? item.members.map(member => [key(member), memberName(member, item.type)]) : item.unitIds.map(id => [id, unitName(id, config)]);
        const describe = (item, config) => [`名称：${item.name}`, ...(collection === "units" ? [`类型：${typeLabel(item.type)}`] : []), ...entries(item, config).map(([, name]) => name)].join("\n");
        for (const id of new Set([...oldItems.keys(), ...newItems.keys()])) {
          const old = oldItems.get(id), current = newItems.get(id), object = `${label}：${itemTitle(current || old)}`;
          const add = (field, from, to) => changes.push([object, field, from, to]);
          if (!old) { add("新增", "—", describe(current, after)); continue; }
          if (!current) { add("删除", describe(old, before), "—"); continue; }
          if (old.name !== current.name) add("名称", old.name, current.name);
          if (old.type !== current.type) add("单位类型", typeLabel(old.type), typeLabel(current.type));
          const oldEntries = new Map(entries(old, before)), newEntries = new Map(entries(current, after));
          for (const [key, name] of oldEntries) if (!newEntries.has(key)) add("移除成员", name, "—");
          for (const [key, name] of newEntries) if (!oldEntries.has(key)) add("加入成员", "—", name);
          if (oldEntries.size === newEntries.size && [...oldEntries.keys()].every(key => newEntries.has(key))
            && JSON.stringify([...oldEntries.keys()]) !== JSON.stringify([...newEntries.keys()])) add("成员顺序", [...oldEntries.values()].join("\n"), [...newEntries.values()].join("\n"));
        }
      }
      return changes;
    }
    function showSavePreview() {
      if (busy || !dirty()) return;
      if (hasOtherChanges()) { showNotice("请先保存或放弃其他页面的修改，再保存活动组。", true); return; }
      const request = { ...envelope(), config: clone(draft) }, changes = configChanges(JSON.parse(baseline), request.config);
      const dialog = el("dialog", null, "confirm-dialog activity-group-dialog"); dialog.setAttribute("aria-label", "活动组配置变更预览");
      dialog.append(el("h2", "活动组配置变更预览"), el("p", `共 ${changes.length} 项变更。确认后保存活动组和活动单位配置。`));
      const scroll = el("div", null, "activity-group-preview-scroll activity-group-config-preview"), table = el("table"), head = el("thead"), header = el("tr"), body = el("tbody");
      ["对象", "变更", "修改前", "修改后"].forEach(text => { const cell = el("th", text); cell.scope = "col"; header.append(cell); }); head.append(header);
      changes.forEach(change => { const row = el("tr"); change.forEach(text => row.append(el("td", text))); body.append(row); });
      table.append(head, body); scroll.append(table); dialog.append(scroll);
      const errorMessage = el("p", "", "notice error hidden"); errorMessage.setAttribute("role", "alert"); dialog.append(errorMessage);
      let saving = false;
      const close = button("取消", () => dialog.close()), submit = button("确认保存", async () => {
        if (saving) return;
        saving = true; busy = true; submit.disabled = close.disabled = true; errorMessage.classList.add("hidden"); render();
        try { await save(request); dialog.close(); }
        catch (error) { errorMessage.textContent = error.message; errorMessage.classList.remove("hidden"); }
        finally { saving = false; busy = false; submit.disabled = close.disabled = false; render(); }
      }, "button primary");
      const actions = el("div", null, "save-actions"); actions.append(close, submit); dialog.append(actions);
      dialog.addEventListener("cancel", event => { if (saving) event.preventDefault(); });
      dialog.addEventListener("close", () => dialog.remove()); document.body.append(dialog); dialog.showModal();
    }
    function add() {
      const id = `${mode === "groups" ? "group" : "unit"}-${crypto.randomUUID()}`;
      draft[mode].push(mode === "groups" ? { id, name: "新活动组", unitIds: [] } : { id, name: "新活动单位", type: 1, members: [] });
      selected = id; search = ""; if (mode === "units") unitTypeFilter = ""; render();
    }
    function remove(items) {
      if (!items.length) return;
      const ids = new Set(items.map(item => item.id));
      const affected = mode === "units" ? draft.groups.filter(group => group.unitIds.some(id => ids.has(id))) : [];
      const emptyCount = affected.filter(group => group.unitIds.every(id => ids.has(id))).length;
      const impact = affected.length ? `将从 ${affected.length} 个活动组移除所选单位${emptyCount ? `，并删除因此变空的 ${emptyCount} 个活动组` : ""}。` : "";
      const target = items.length === 1 ? `“${itemTitle(items[0])}”` : `所选的 ${items.length} 个${mode === "units" ? "活动单位" : "活动组"}`;
      if (!window.confirm(`删除${target}？${impact}保存配置后生效。`)) return;
      draft[mode] = draft[mode].filter(row => !ids.has(row.id));
      if (mode === "units") draft.groups = draft.groups.flatMap(group => {
        if (!group.unitIds.some(id => ids.has(id))) return [group];
        const unitIds = group.unitIds.filter(id => !ids.has(id));
        return unitIds.length ? [{ ...group, unitIds }] : [];
      });
      if (ids.has(selected)) selected = "";
      render();
    }
    function updateSaveState() {
      const saveButton = root.querySelector("[data-group-save]");
      if (saveButton) saveButton.disabled = busy || !dirty();
      const summary = root.querySelector("[data-group-summary]");
      if (summary) summary.textContent = dirty() ? "有未保存的活动组修改" : "活动组配置已保存";
      root.querySelectorAll("[data-group-schedule]").forEach(node => { node.disabled = busy || dirty(); });
    }
    function render() {
      const previousSidebar = root.querySelector(".activity-group-sidebar");
      if (previousSidebar) sidebarScroll[previousSidebar.dataset.mode] = previousSidebar.scrollTop;
      const previousDetail = root.querySelector(".activity-group-detail");
      const detailPosition = { selection: previousDetail?.dataset.selection, top: previousDetail?.scrollTop || 0 };
      root.replaceChildren();
      if (!draft) return;
      for (const value of ["groups", "units"]) {
        const available = new Set(draft[value].map(item => item.id));
        for (const id of checkedIDs[value]) if (!available.has(id)) checkedIDs[value].delete(id);
      }
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
      toolbar.append(button(mode === "groups" ? "新建活动组" : "新建活动单位", add, "button ghost activity-group-add")); root.append(toolbar);
      const layout = el("div", null, "activity-group-layout"), sidebar = el("aside", null, "activity-group-sidebar"), detail = el("section", null, "activity-group-detail");
      sidebar.dataset.mode = mode;
      const searchInput = input(search, value => { search = value; sidebar.scrollTop = 0; renderList(); }, "search"); searchInput.placeholder = "搜索名称或 ID"; searchInput.setAttribute("aria-label", "搜索活动组或单位");
      const filteredItems = () => {
        const type = mode === "units" ? Number(unitTypeFilter) : 0;
        return draft[mode].filter(item => (!type || item.type === type)
          && `${item.id} ${item.name} ${itemTitle(item)}`.toLowerCase().includes(search.toLowerCase()));
      };
      const batchActions = el("div", null, "activity-group-batch-actions");
      const selectAll = el("input", null, "activity-group-select-all"); selectAll.type = "checkbox";
      selectAll.addEventListener("change", () => {
        for (const item of filteredItems()) {
          if (selectAll.checked) checkedIDs[mode].add(item.id); else checkedIDs[mode].delete(item.id);
        }
        renderList();
      });
      const selectAllLabel = el("label"); selectAllLabel.append(selectAll, el("span", "全选当前结果"));
      const deleteSelected = button("", () => remove(draft[mode].filter(item => checkedIDs[mode].has(item.id))), "button ghost activity-group-batch-delete");
      batchActions.append(selectAllLabel, deleteSelected);
      const list = el("div", null, "activity-group-list"); sidebar.append(searchInput);
      if (mode === "units") {
        const typeFilter = select([["", "全部类型"], ...unitTypes], unitTypeFilter, value => {
          unitTypeFilter = value; sidebar.scrollTop = 0; renderList();
        }, "类型筛选");
        sidebar.append(field("类型筛选", typeFilter));
      }
      sidebar.append(batchActions, list);
      function updateSelectionState() {
        const rows = filteredItems(), count = rows.filter(item => checkedIDs[mode].has(item.id)).length;
        selectAll.checked = rows.length > 0 && count === rows.length;
        selectAll.indeterminate = count > 0 && count < rows.length;
        selectAll.disabled = busy || !rows.length;
        deleteSelected.textContent = `删除所选（${checkedIDs[mode].size}）`;
        deleteSelected.disabled = busy || !checkedIDs[mode].size;
      }
      function renderList() {
        const scrollTop = sidebar.scrollTop;
        list.replaceChildren();
        const rows = filteredItems();
        if (!rows.length) list.append(el("p", "暂无匹配内容"));
        rows.forEach(item => {
          const row = el("div", null, `activity-group-list-row${selected === item.id ? " active" : ""}`);
          const choose = button(itemTitle(item), () => { selected = item.id; renderList(); renderDetail(); updateSaveState(); }, "activity-group-list-item");
          choose.title = item.id; choose.setAttribute("aria-pressed", String(selected === item.id));
          choose.append(el("small", mode === "groups" ? `${item.unitIds.length} 个单位` : `${typeLabel(item.type)} · ${visibleMembers(item).length} 个条目`));
          const checkbox = el("input", null, "activity-group-list-check"); checkbox.type = "checkbox";
          checkbox.checked = checkedIDs[mode].has(item.id); checkbox.disabled = busy;
          checkbox.setAttribute("aria-label", `勾选${mode === "groups" ? "活动组" : "活动单位"} ${itemTitle(item)}`);
          checkbox.addEventListener("change", () => {
            if (checkbox.checked) checkedIDs[mode].add(item.id); else checkedIDs[mode].delete(item.id);
            updateSelectionState();
          });
          row.append(checkbox, choose); list.append(row);
        });
        sidebar.scrollTop = scrollTop;
        updateSelectionState();
      }
      renderList();
      function renderDetail() {
        const selection = `${mode}:${selected}`, scrollTop = detail.dataset.selection === selection ? detail.scrollTop : 0;
        detail.dataset.selection = selection;
        detail.replaceChildren();
        const item = draft[mode].find(row => row.id === selected);
        if (!item) detail.append(el("p", "选择已有内容，或新建活动组 / 活动单位。", "activity-group-note"));
        else {
          const name = input(itemName(item), value => { item.name = value; renderList(); updateSaveState(); });
          name.maxLength = 200;
          const header = el("div", null, `activity-group-header${mode === "units" ? " has-type" : ""}`);
          if (mode === "units") {
            const type = select(unitTypes, item.type, value => {
              const next = { ...item, type: Number(value) };
              const compatible = item.members.filter(member => memberAllowed(optionFor(member), next));
              const removed = item.members.length - compatible.length;
              if (removed && !window.confirm(`切换类型会移除 ${removed} 个不适用的条目，是否继续？`)) { type.value = item.type; return; }
              item.type = next.type; item.members = compatible; render();
            }, "单位类型");
            header.append(field("单位类型", type));
          }
          header.append(field("名称", name), button("删除", () => remove([item]))); detail.append(header);
          if (mode === "units") renderUnit(detail, item); else renderGroup(detail, item);
        }
        detail.scrollTop = scrollTop;
      }
      renderDetail();
      layout.append(sidebar, detail); root.append(layout);
      sidebar.scrollTop = sidebarScroll[mode];
      if (detailPosition.selection === detail.dataset.selection) detail.scrollTop = detailPosition.top;
      const footer = el("div", null, "savebar"); const summary = el("strong"); summary.dataset.groupSummary = "";
      const actions = el("div", null, "save-actions");
      const saveButton = button("保存活动组配置", showSavePreview, "button primary"); saveButton.dataset.groupSave = "";
      actions.append(button("放弃修改", () => {
        if (!window.confirm("放弃全部尚未保存的活动组和活动单位修改？")) return;
        draft = clone(data.catalog.config); clearChecked(); render();
      }), saveButton); footer.append(summary, actions); root.append(footer); updateSaveState();
    }
    function renderUnit(detail, unit) {
      if (unit.type === 3) detail.append(el("p", "Event Gacha 仅可添加当前 Variation 副本对应的票池；铜、银、金票池分别配置，整组改时会更新已添加的条目。", "activity-group-note"));
      for (const [label, kinds] of sectionsByType[unit.type] || []) {
        const section = el("section", null, "activity-group-member-section"); section.dataset.memberSection = kinds[0];
        const members = visibleMembers(unit).filter(member => kinds.includes(member.kind));
        const isPrimary = kinds[0] === "premium" || kinds[0] === "chapter";
        const isSingle = isPrimary || kinds[0] === "shop";
        const heading = el("div", null, "activity-group-section-heading"); heading.append(el("h3", label));
        if (!isSingle) heading.append(el("span", String(members.length), "activity-group-section-count")); section.append(heading);
        if (isSingle) {
          const options = data.catalog.options.filter(option => kinds.includes(option.kind) && memberAllowed(option, unit));
          const picker = select([["", isPrimary ? `选择${label}` : "不关联"], ...options.map(option => [key(option), title(option)])], members.length === 1 ? key(members[0]) : "", value => {
            const option = options.find(option => key(option) === value);
            if (isPrimary && !option) return;
            unit.members = unit.members.filter(member => !(isPrimary ? member.kind === "premium" || member.kind === "chapter" : kinds.includes(member.kind))
              && !(isPrimary && unit.type === 3 && member.kind === "event" && optionFor(member)?.relatedChapterId !== option.id));
            if (option) unit.members.unshift({ kind: option.kind, id: option.id });
            render();
          }, `选择${label}`);
          picker.children[0].disabled = isPrimary; picker.required = isPrimary; picker.dataset.searchable = "true";
          const selection = el("div", null, "activity-group-single-select"); selection.append(picker); section.append(selection);
        } else if (!members.length) section.append(el("p", "尚未添加条目", "activity-group-empty"));
        const hasPreview = kinds[0] === "banner" || kinds[0] === "premium";
        const entries = el("div", null, kinds[0] === "banner" ? "activity-group-banner-grid" : "");
        for (const member of members) {
          const rowClass = member.kind === "banner" ? "activity-group-banner-card" : member.kind === "premium" ? "activity-group-entry activity-group-premium-row" : "activity-group-entry";
          const option = optionFor(member), row = el("div", null, rowClass), copy = el("div", null, "activity-group-entry-copy");
          if (hasPreview) row.append(renderBannerPreview(option));
          copy.append(el("strong", title(member)), el("small", option ? `${formatTime(option.startDatetime)} → ${formatTime(option.endDatetime)}` : "引用已失效"));
          if (unit.type === 1 && member.kind === "term") {
            const conversion = optionFor({ kind: "medal", id: member.id });
            if (conversion) copy.append(el("small", `自动转换时间：${formatTime(conversion.endDatetime)}`));
          }
          row.append(copy);
          if (!isSingle) row.append(button("移除", () => {
            unit.members = unit.members.filter(item => key(item) !== key(member) && !(unit.type === 1 && member.kind === "term" && item.kind === "medal" && item.id === member.id));
            render();
          })); entries.append(row);
        }
        section.append(entries);
        if (isSingle) { detail.append(section); continue; }
        const options = data.catalog.options.filter(option => kinds.includes(option.kind) && memberAllowed(option, unit) && !members.some(member => key(member) === key(option)));
        let memberKey = "";
        const picker = select([["", `选择${label}`], ...options.map(option => [key(option), title(option)])], "", value => { memberKey = value; addButton.disabled = busy || !value; }, `添加${label}`);
        picker.dataset.searchable = "true";
        const addButton = button("添加条目", () => {
          const option = options.find(option => key(option) === memberKey);
          if (option) {
            unit.members.push({ kind: option.kind, id: option.id });
            const conversion = { kind: "medal", id: option.id };
            if (unit.type === 1 && option.kind === "term" && optionFor(conversion) && !unit.members.some(member => key(member) === key(conversion))) unit.members.push(conversion);
            render();
          }
        }); addButton.disabled = true;
        const addRow = el("div", null, "activity-group-section-add"); addRow.append(picker, addButton); section.append(addRow); detail.append(section);
      }
    }
    function renderGroup(detail, group) {
      detail.append(el("h3", "活动单位"));
      const unitList = el("div", null, "activity-group-unit-list"), table = el("table", null, "activity-group-unit-table");
      const head = el("thead"), header = el("tr"), body = el("tbody");
      for (const label of ["ID", "名称", "类型", "操作"]) { const cell = el("th", label); cell.scope = "col"; header.append(cell); }
      head.append(header);
      group.unitIds.forEach(id => {
        const unit = draft.units.find(item => item.id === id), row = el("tr"), actions = el("td");
        const buttons = el("div", null, "activity-group-unit-actions");
        buttons.append(button("编辑单位", () => { mode = "units"; selected = id; search = ""; unitTypeFilter = ""; render(); }), button("移除", () => { group.unitIds = group.unitIds.filter(value => value !== id); render(); }));
        actions.append(buttons);
        row.append(el("td", id.replace(/^(chapter|premium):/, "")), el("td", unit ? itemName(unit) : "引用已失效"), el("td", unit ? typeLabel(unit.type) : "—"), actions); body.append(row);
      });
      if (!group.unitIds.length) { const row = el("tr"), cell = el("td", "尚未加入活动单位", "activity-group-empty"); cell.colSpan = 4; row.append(cell); body.append(row); }
      table.append(head, body); unitList.append(table); detail.append(unitList);
      let unitID = ""; const addRow = el("div", null, "activity-group-section-add activity-group-unit-add");
      addRow.append(select([["", "选择活动单位"], ...draft.units.filter(unit => !group.unitIds.includes(unit.id)).map(unit => [unit.id, itemTitle(unit)])], "", value => { unitID = value; }, "组合活动单位"), button("加入活动组", () => { if (unitID) { group.unitIds.push(unitID); render(); } })); detail.append(addRow);
      detail.append(el("h3", "整体修改时间"), el("p", "使用本机时区。副本、Premium Gacha、Banner、NaviCutIn 和 Tip 使用活动起止时间；活动任务、Event Gacha、兑换商店与道具 / 碎片有效期延长至结束后 48 小时。请先保存配置，再预览改时。", "activity-group-note"));
      const source = draft.units.filter(unit => group.unitIds.includes(unit.id)).flatMap(unit => unit.members).find(member => member.kind === "chapter" || member.kind === "premium");
      const option = source && optionFor(source);
      const start = input(localInput(option?.startDatetime), () => {}, "datetime-local"), end = input(localInput(option?.endDatetime), () => {}, "datetime-local");
      start.step = "1"; end.step = "1";
      const controls = el("div", null, "activity-group-schedule");
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
      dialog.append(el("h2", `${itemTitle(group)}：改时预览`), el("p", `${changes.length} 个字段将被修改。共享成员只更新一次，也会影响使用这些成员的其他活动组。`));
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
    return { load, render, dirty, reset: () => { data = null; draft = null; baseline = ""; clearChecked(); unitTypeFilter = ""; } };
  };
})();
