(() => {
  "use strict";
  const visibleRows = 8, instances = new Map();
  const disabled = option => Boolean(option.disabled || option.parentElement?.disabled);
  let current = null, serial = 0, scheduled = false;
  const labelFor = option => {
    const title = option.textContent.trim(), id = option.value;
    return id && !title.includes(id) ? `${title} · ${id}` : title;
  };
  function enhance(select, config) {
    if (select.dataset.searchable === "false" || select.multiple) return select;
    if (instances.has(select)) {
      const instance = instances.get(select); if (config) instance.configure(config); return instance.wrapper;
    }
    config = config || {};
    const rowHeight = config.renderOption ? 44 : 38;
    const wrapper = document.createElement("span"); wrapper.className = "search-select";
    select.before(wrapper); wrapper.append(select); select.classList.add("search-select-native");
    const input = document.createElement("input"); input.type = "text"; input.autocomplete = "off";
    input.setAttribute("role", "combobox"); input.setAttribute("aria-autocomplete", "list"); input.setAttribute("aria-expanded", "false");
    let title = config.ariaLabel || config.placeholder || select.getAttribute("aria-label") || [...(select.labels || [])].flatMap(label => [...label.childNodes].filter(n => n.nodeType === 3).map(n => n.textContent.trim())).join(" ") || select.title || "搜索并选择";
    input.setAttribute("aria-label", title); input.placeholder = "输入标题或 ID 搜索";
    const arrow = document.createElement("button"); arrow.type = "button"; arrow.tabIndex = -1; arrow.textContent = "▾"; arrow.setAttribute("aria-label", `展开${title}`);
    wrapper.append(input, arrow);
    let options = [], matches = [], active = -1, menu = null, viewport = null, spacer = null, query = "", optionSource = null;
    const listID = `search-select-list-${++serial}`; input.setAttribute("aria-controls", listID);
    function sync() {
      if (input.disabled !== select.disabled) input.disabled = select.disabled;
      if (arrow.disabled !== select.disabled) arrow.disabled = select.disabled;
      if (input.required !== select.required) input.required = select.required;
      input.title = select.title;
      input.placeholder = config.placeholder || "输入标题或 ID 搜索";
      if (!config.options) options = [...select.options].filter(o => !o.hidden).map(option => ({ option, label: labelFor(option), search: `${labelFor(option)} ${option.dataset.search || option.dataset.searchText || ""} ${option.parentElement?.tagName === "OPTGROUP" ? option.parentElement.label : ""}`.toLowerCase() }));
      if (!menu) input.value = select.selectedOptions.length ? labelFor(select.selectedOptions[0]) : "";
      else if (select.disabled) close(); else filter();
    }
    function close() {
      menu?.remove(); menu = viewport = spacer = null;
      input.setAttribute("aria-expanded", "false"); input.removeAttribute("aria-activedescendant");
      input.value = select.selectedOptions.length ? labelFor(select.selectedOptions[0]) : "";
      if (current?.input === input) current = null;
    }
    function position() {
      if (!menu) return;
      const rect = input.getBoundingClientRect(), width = Math.min(Math.max(rect.width, 320), innerWidth - 20);
      menu.style.width = `${width}px`; menu.style.left = `${Math.max(10, Math.min(rect.left, innerWidth - width - 10))}px`;
      const height = Math.min(Math.max(matches.length, 1), visibleRows) * rowHeight;
      const below = innerHeight - rect.bottom - 12, above = rect.top - 12, flip = below < height && above > below;
      viewport.style.height = `${Math.min(height, Math.max(76, flip ? above : below))}px`;
      menu.style.top = `${flip ? Math.max(6, rect.top - viewport.offsetHeight - 2) : rect.bottom + 2}px`;
    }
    function paint() {
      if (!menu) return;
      spacer.replaceChildren(); spacer.style.height = `${Math.max(matches.length, 1) * rowHeight}px`;
      if (!matches.length) { const empty = document.createElement("div"); empty.className = "search-select-empty"; empty.textContent = config.emptyText || "没有匹配选项"; spacer.append(empty); input.removeAttribute("aria-activedescendant"); return; }
      const from = Math.max(0, Math.floor(viewport.scrollTop / rowHeight) - 2), to = Math.min(matches.length, from + visibleRows + 5);
      for (let i = from; i < to; i++) {
        const item = matches[i], row = document.createElement("div"); row.id = `${listID}-${i}`;
        row.className = `search-select-option${i === active ? " active" : ""}${disabled(item.option) ? " disabled" : ""}`;
        row.setAttribute("role", "option"); row.setAttribute("aria-selected", String(item.option.value === select.value)); row.setAttribute("aria-disabled", String(disabled(item.option)));
        row.setAttribute("aria-posinset", String(i + 1)); row.setAttribute("aria-setsize", String(matches.length));
        row.style.top = `${i * rowHeight}px`; row.style.height = `${rowHeight}px`; row.title = item.label;
        const visual = config.renderOption?.(item.source || item);
        if (visual) { row.classList.add("with-visual"); row.append(visual); } else row.textContent = item.group ? `${item.group} · ${item.label}` : item.label;
        row.addEventListener("pointerdown", event => { event.preventDefault(); choose(i); }); spacer.append(row);
      }
      if (active >= from && active < to) input.setAttribute("aria-activedescendant", `${listID}-${active}`); else input.removeAttribute("aria-activedescendant");
    }
    function filter() {
      const tokens = query.toLowerCase().trim().split(/\s+/).filter(Boolean);
      matches = options.filter(item => tokens.every(token => item.search.includes(token)));
      const groups = new Map();
      for (const item of matches) {
        const key = item.option.dataset.searchGroup; if (!key) continue;
        const old = groups.get(key);
        if (!old || item.option.value === query.trim() || (old.option.value !== query.trim() && item.option.selected)) groups.set(key, item);
      }
      matches = matches.filter(item => !item.option.dataset.searchGroup || groups.get(item.option.dataset.searchGroup) === item);
      const normalized = query.toLowerCase().trim();
      if (normalized) {
        const rank = item => item.option.value.toLowerCase() === normalized ? 0 : item.option.value.toLowerCase().startsWith(normalized) ? 1 : item.label.toLowerCase().startsWith(normalized) ? 2 : 3;
        matches.sort((a, b) => rank(a) - rank(b));
      }
      active = matches.findIndex(item => item.option.value === select.value && !disabled(item.option));
      if (active < 0) active = matches.findIndex(item => !disabled(item.option));
      if (viewport) { viewport.scrollTop = Math.max(0, active - 2) * rowHeight; position(); paint(); }
    }
    function open() {
      if (select.disabled || menu) return;
      current?.close(); query = ""; sync();
      if (config.options) {
        const source = typeof config.options === "function" ? config.options() : config.options;
        if (source !== optionSource) {
          optionSource = source;
          options = source.map(entry => {
            const option = { value: String(entry.value), textContent: String(entry.label), disabled: Boolean(entry.disabled), dataset: {} };
            return { option, source: entry, group: entry.group, label: labelFor(option), search: `${entry.value} ${entry.label} ${entry.searchText || ""} ${entry.group || ""}`.toLowerCase() };
          });
        }
      }
      menu = document.createElement("div"); menu.className = "search-select-menu";
      viewport = document.createElement("div"); viewport.id = listID; viewport.className = "search-select-viewport"; viewport.setAttribute("role", "listbox"); viewport.setAttribute("aria-label", title);
      spacer = document.createElement("div"); spacer.className = "search-select-spacer"; viewport.append(spacer); menu.append(viewport);
      (select.closest("dialog[open]") || document.body).append(menu);
      viewport.addEventListener("scroll", paint); input.setAttribute("aria-expanded", "true"); current = { input, close, position, wrapper, menu }; filter();
    }
    function choose(index) {
      const item = matches[index]; if (!item || disabled(item.option) || select.disabled) return;
      if (item.source) { const option = document.createElement("option"); option.value = item.option.value; option.textContent = item.option.textContent; select.replaceChildren(option); }
      select.value = item.option.value; close();
      select.dispatchEvent(new Event("input", { bubbles: true })); select.dispatchEvent(new Event("change", { bubbles: true }));
    }
    function move(direction) {
      if (!menu) open();
      for (let i = active + direction; i >= 0 && i < matches.length; i += direction) {
        if (disabled(matches[i].option)) continue;
        active = i;
        if (i * rowHeight < viewport.scrollTop) viewport.scrollTop = i * rowHeight;
        else if ((i + 1) * rowHeight > viewport.scrollTop + viewport.clientHeight) viewport.scrollTop = (i + 1) * rowHeight - viewport.clientHeight;
        paint(); break;
      }
    }
    input.addEventListener("focus", () => { open(); input.select(); }); input.addEventListener("click", open);
    input.addEventListener("input", () => { const typed = input.value; if (!menu) open(); input.value = typed; query = typed; filter(); });
    input.addEventListener("blur", () => { setTimeout(() => { if (document.activeElement !== input) close(); }, 0); });
    input.addEventListener("keydown", event => {
      if (event.key === "ArrowDown" || event.key === "ArrowUp") { event.preventDefault(); move(event.key === "ArrowDown" ? 1 : -1); }
      else if (event.key === "Enter" && menu) { event.preventDefault(); choose(active); }
      else if (event.key === "Escape" && menu) { event.preventDefault(); event.stopPropagation(); close(); }
      else if (event.key === "Tab") close();
    });
    arrow.addEventListener("pointerdown", event => { event.preventDefault(); if (menu) close(); else { input.focus(); open(); } });
    select.addEventListener("change", sync);
    // Preserve the value API used by the existing native-select editors.
    const names = ["value", "selectedIndex"];
    for (const name of names) {
      const descriptor = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, name);
      Object.defineProperty(select, name, { configurable: true, get() { return descriptor.get.call(this); }, set(value) { descriptor.set.call(this, value); sync(); } });
    }
    instances.set(select, { wrapper, sync, configure(next) { config = next; optionSource = null; title = config.ariaLabel || config.placeholder || title; input.setAttribute("aria-label", title); close(); sync(); }, dispose() { close(); select.removeEventListener("change", sync); for (const name of names) delete select[name]; } }); sync();
    return wrapper;
  }
  function refresh() {
    for (const [select, instance] of instances) { if (!select.isConnected) { instance.dispose(); instances.delete(select); } else instance.sync(); }
    document.querySelectorAll("select").forEach(select => {
      if (select.dataset.searchable === "true" || (select.options.length >= 12 && [...select.options].some(option => /[^\d\s.,+-]/.test(option.textContent)))) enhance(select);
    });
  }
  function schedule() { if (!scheduled) { scheduled = true; requestAnimationFrame(() => { scheduled = false; refresh(); }); } }
  window.AdminSearchSelect = { enhance, refresh };
  const observer = new MutationObserver(records => { if (records.some(record => !(record.target.nodeType === 1 ? record.target : record.target.parentElement)?.closest(".search-select-menu"))) schedule(); });
  observer.observe(document.documentElement, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ["disabled", "selected", "required", "data-searchable"] });
  document.addEventListener("pointerdown", event => { if (current && !current.wrapper.contains(event.target) && !current.menu.contains(event.target)) current.close(); });
  document.addEventListener("scroll", () => current?.position(), true); window.addEventListener("resize", () => current?.position());
  document.addEventListener("reset", () => setTimeout(refresh, 0)); schedule();
})();
