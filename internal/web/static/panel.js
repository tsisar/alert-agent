/* Alert Agent web UI behaviour.
   Progressive: every control here enhances markup that already works. */
(function () {
	"use strict";

	var doc = document;

	function $(sel, root) {
		return (root || doc).querySelector(sel);
	}

	function $$(sel, root) {
		return Array.prototype.slice.call((root || doc).querySelectorAll(sel));
	}

	function escapeHTML(s) {
		return String(s).replace(/[&<>"']/g, function (c) {
			return {
				"&": "&amp;",
				"<": "&lt;",
				">": "&gt;",
				'"': "&quot;",
				"'": "&#39;"
			}[c];
		});
	}

	var SVG_OPEN =
		'<svg class="icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">';
	var ICON_CHECK = SVG_OPEN + '<circle cx="12" cy="12" r="9"/><path d="m8 12.5 2.7 2.7L16.5 9.5"/></svg>';
	var ICON_ALERT =
		SVG_OPEN +
		'<path d="M10.3 3.9 2.4 17.6A2 2 0 0 0 4.1 20.6h15.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/><path d="M12 9.5v4.2M12 16.8v.2"/></svg>';
	var ICON_SPIN =
		'<svg class="icon spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M12 3.2a8.8 8.8 0 1 0 8.8 8.8"/></svg>';
	var ICON_X = SVG_OPEN + '<path d="M6 6l12 12M18 6L6 18"/></svg>';
	var ICON_CHEVRON = SVG_OPEN + '<path d="m9 5 7 7-7 7"/></svg>';

	/* --- Theme ---------------------------------------------------------- */

	function themeLabel(theme) {
		return theme === "dark" ? "Switch to light theme" : "Switch to dark theme";
	}

	function initTheme() {
		var btn = $("[data-theme-toggle]");
		if (!btn || btn.dataset.bound) return;
		btn.dataset.bound = "1";
		var root = doc.documentElement;
		btn.setAttribute("aria-label", themeLabel(root.getAttribute("data-theme")));
		btn.addEventListener("click", function () {
			var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
			root.setAttribute("data-theme", next);
			btn.setAttribute("aria-label", themeLabel(next));
			try {
				localStorage.setItem("theme", next);
			} catch (e) {}
		});
	}

	/* --- Auto-resizing textareas ---------------------------------------- */

	function resize(el) {
		el.style.height = "auto";
		el.style.height = el.scrollHeight + 2 + "px";
	}

	function initAutoResize(root) {
		$$("textarea[data-autoresize]", root).forEach(function (el) {
			if (!el.dataset.resizeBound) {
				el.addEventListener("input", function () {
					resize(el);
				});
				el.dataset.resizeBound = "1";
			}
			resize(el);
		});
	}

	/* --- Scenario list: filter ------------------------------------------ */

	function initFilter() {
		var input = $("[data-filter-input]");
		var field = $("[data-field]");
		if (!input || !field || input.dataset.bound) return;
		input.dataset.bound = "1";

		var countEl = $("[data-filter-count]");
		var emptyEl = $("[data-filter-empty]");

		function apply() {
			var q = input.value.trim().toLowerCase();
			var rows = $$(".scn-row", field);
			var shown = 0;
			rows.forEach(function (row) {
				var hit = !q || (row.dataset.search || "").indexOf(q) !== -1;
				row.hidden = !hit;
				if (hit) shown++;
			});
			// While filtering, drop group headers and notes whose rows are all hidden.
			$$("tbody", field).forEach(function (group) {
				var any = $$(".scn-row", group).some(function (r) {
					return !r.hidden;
				});
				$$(".group-head, .group-note", group).forEach(function (el) {
					el.hidden = !!q && !any;
				});
			});
			field.hidden = !!q && shown === 0;
			if (countEl) {
				var n = rows.length;
				countEl.textContent = q
					? shown + " of " + n + " shown"
					: n + (n === 1 ? " scenario" : " scenarios");
			}
			if (emptyEl) emptyEl.hidden = shown !== 0 || !q;
		}

		input.addEventListener("input", apply);
		input.addEventListener("keydown", function (e) {
			if (e.key === "Escape" && input.value) {
				e.stopPropagation();
				input.value = "";
				apply();
				return;
			}
			// Typing narrows to one scenario; Enter opens it without the mouse.
			if (e.key === "Enter") {
				e.preventDefault();
				var visible = $$(".scn-row", field).filter(function (row) {
					return !row.hidden;
				});
				if (visible.length === 1) {
					var link = $(".scn-name a", visible[0]);
					if (link) window.location.href = link.href;
				}
			}
		});

		apply();
	}

	function initSlashKey() {
		doc.addEventListener("keydown", function (e) {
			if (e.key !== "/" || e.metaKey || e.ctrlKey || e.altKey) return;
			var tag = (doc.activeElement && doc.activeElement.tagName) || "";
			if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
			var input = $("[data-filter-input]");
			if (!input) return;
			e.preventDefault();
			input.focus();
			input.select();
		});
	}

	/* --- Scenario list: test an alert ----------------------------------- */

	function initProbe() {
		var form = $("[data-probe-form]");
		if (!form || form.dataset.bound) return;
		form.dataset.bound = "1";

		var input = $("[data-probe-input]", form);
		var out = $("[data-probe-result]");

		function clearLive() {
			$$(".scn-row[data-live]").forEach(function (row) {
				row.removeAttribute("data-live");
			});
		}

		function render(outcome, html) {
			if (!out) return;
			out.hidden = false;
			out.setAttribute("data-outcome", outcome);
			var icon =
				outcome === "match" ? ICON_CHECK : outcome === "pending" ? ICON_SPIN : ICON_ALERT;
			out.innerHTML = icon + "<div>" + html + "</div>";
		}

		form.addEventListener("submit", function (e) {
			e.preventDefault();
			var labels = input.value.trim();
			clearLive();
			if (!labels) {
				render(
					"error",
					"Enter at least one <span class='mono'>key=value</span> label, for example <span class='mono'>alertname=HighCPU</span>."
				);
				input.focus();
				return;
			}
			render("pending", "<span class='probe-result-note'>Testing…</span>");

			fetch("/api/match-probe", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ labels: labels })
			})
				.then(function (res) {
					if (!res.ok) throw new Error("the server answered " + res.status);
					return res.json();
				})
				.then(function (data) {
					if (data.error) {
						render("error", escapeHTML(data.error));
						return;
					}
					if (!data.matched) {
						render(
							"none",
							"<strong>No scenario handles this alert.</strong> <span class='probe-result-note'>Nothing matches these labels and there is no fallback scenario, so the alert would be dropped.</span>"
						);
						return;
					}
					// The list is re-rendered by reorder and delete, so look it up now.
					var row = $('.scn-row[data-id="' + data.id + '"]');
					var hiddenByFilter = false;
					if (row) {
						row.setAttribute("data-live", "true");
						// The filter owns row.hidden. Un-hiding here would leave the
						// count contradicting the list, so say so instead.
						hiddenByFilter = row.hidden;
						if (!hiddenByFilter) {
							row.scrollIntoView({ block: "nearest", behavior: "smooth" });
						}
					}
					var rankEl = row && row.querySelector("[data-rank]");
					var where = rankEl ? " (#" + rankEl.textContent.trim() + ")" : "";
					var why =
						data.reason === "catch-all"
							? "No scenario matched these labels, so the fallback handles it."
							: "Matched on <span class='mono'>" + escapeHTML(data.matchedOn) + "</span>.";
					if (hiddenByFilter) {
						why += " It is hidden by the current filter.";
					}
					render(
						"match",
						"Handled by <strong>" +
							escapeHTML(data.name) +
							"</strong>" +
							where +
							". <span class='probe-result-note'>" +
							why +
							"</span>"
					);
				})
				.catch(function (err) {
					render("error", "The test could not run: " + escapeHTML(String(err.message || err)) + ".");
				});
		});
	}

	/* --- Scenario list: inline delete confirm --------------------------- */

	function closeConfirm(row, restoreFocus) {
		if (!row) return;
		row.removeAttribute("data-confirming");
		if (restoreFocus) {
			var ask = $("[data-delete-ask]", row);
			if (ask) ask.focus();
		}
	}

	function initDeleteConfirm() {
		doc.addEventListener("click", function (e) {
			var ask = e.target.closest("[data-delete-ask]");
			if (ask) {
				var row = ask.closest(".scn-row");
				if (!row) return;
				$$('.scn-row[data-confirming="true"]').forEach(function (other) {
					if (other !== row) closeConfirm(other, false);
				});
				row.setAttribute("data-confirming", "true");
				// The pressed button is now hidden, so move focus deliberately
				// instead of letting it fall to <body>.
				var confirm = $("[data-delete-confirm]", row);
				if (confirm) confirm.focus();
				return;
			}
			var cancel = e.target.closest("[data-delete-cancel]");
			if (cancel) {
				closeConfirm(cancel.closest(".scn-row"), true);
			}
		});

		doc.addEventListener("keydown", function (e) {
			if (e.key !== "Escape") return;
			var row =
				doc.activeElement && doc.activeElement.closest
					? doc.activeElement.closest('.scn-row[data-confirming="true"]')
					: null;
			if (!row) row = $('.scn-row[data-confirming="true"]');
			if (row) {
				e.stopPropagation();
				closeConfirm(row, true);
			}
		});
	}

	/* --- Editor: match label pairs -------------------------------------- */

	function syncPairs() {
		var store = $("[data-pairs-store]");
		var list = $("[data-pairs]");
		if (!store || !list) return;
		var lines = [];
		$$(".pair", list).forEach(function (row) {
			var k = $("[data-pair-key]", row).value.trim();
			var v = $("[data-pair-value]", row).value.trim();
			if (k) lines.push(k + "=" + v);
		});
		store.value = lines.join("\n");
		var empty = $("[data-pairs-empty]");
		if (empty) empty.hidden = $$(".pair", list).length !== 0;
		var badge = $("[data-pairs-badge]");
		if (badge) {
			badge.textContent = lines.length
				? lines.length + (lines.length === 1 ? " label" : " labels")
				: "Fallback";
		}
	}

	function pairRow(key, value) {
		var row = doc.createElement("div");
		row.className = "pair";
		row.innerHTML =
			'<input type="text" class="mono" data-pair-key placeholder="alertname" aria-label="Label name" spellcheck="false">' +
			'<span class="pair-eq" aria-hidden="true">=</span>' +
			'<input type="text" class="mono" data-pair-value placeholder="HighCPU" aria-label="Label value" spellcheck="false">' +
			'<button type="button" class="btn btn-icon btn-ghost" data-pair-remove aria-label="Remove label" title="Remove label">' +
			ICON_X +
			"</button>";
		$("[data-pair-key]", row).value = key || "";
		$("[data-pair-value]", row).value = value || "";
		return row;
	}

	function initPairs() {
		var list = $("[data-pairs]");
		var store = $("[data-pairs-store]");
		if (!list || !store || list.dataset.bound) return;
		list.dataset.bound = "1";

		store.value
			.split("\n")
			.map(function (l) {
				return l.trim();
			})
			.filter(Boolean)
			.forEach(function (line) {
				var i = line.indexOf("=");
				list.appendChild(
					pairRow(i === -1 ? line : line.slice(0, i), i === -1 ? "" : line.slice(i + 1))
				);
			});

		var add = $("[data-pair-add]");
		if (add) {
			add.addEventListener("click", function () {
				var row = pairRow("", "");
				list.appendChild(row);
				$("[data-pair-key]", row).focus();
				syncPairs();
			});
		}

		list.addEventListener("input", syncPairs);
		list.addEventListener("click", function (e) {
			var rm = e.target.closest("[data-pair-remove]");
			if (!rm) return;
			var row = rm.closest(".pair");
			var next = row.nextElementSibling || row.previousElementSibling;
			row.remove();
			syncPairs();
			markDirty();
			if (next) {
				var focus = $("[data-pair-key]", next);
				if (focus) focus.focus();
			} else if (add) {
				add.focus();
			}
		});

		syncPairs();
	}

	/* --- Editor: tools -------------------------------------------------- */

	var toolState = { available: null, loaded: false, error: null };

	function selectedTools() {
		var store = $("[data-tools-store]");
		if (!store) return [];
		return store.value
			.split(",")
			.map(function (t) {
				return t.trim();
			})
			.filter(Boolean);
	}

	function renderToolChips() {
		var wrap = $("[data-tool-chips]");
		var store = $("[data-tools-store]");
		if (!wrap || !store) return;
		var picked = selectedTools();
		wrap.innerHTML = "";

		if (!picked.length) {
			var all = doc.createElement("span");
			all.className = "chip";
			all.setAttribute("data-open", "true");
			all.textContent = "All tools";
			wrap.appendChild(all);
			return;
		}

		picked.forEach(function (name) {
			var chip = doc.createElement("span");
			chip.className = "chip";
			var stale =
				toolState.loaded &&
				toolState.available &&
				toolState.available.indexOf(name) === -1;
			if (stale) {
				chip.setAttribute("data-stale", "true");
				chip.title = "No connected MCP server currently offers this tool";
			}
			chip.textContent = name;
			var rm = doc.createElement("button");
			rm.type = "button";
			rm.className = "chip-remove";
			rm.setAttribute("aria-label", "Remove " + name);
			rm.innerHTML = ICON_X;
			rm.addEventListener("click", function () {
				var next = selectedTools().filter(function (t) {
					return t !== name;
				});
				store.value = next.join(", ");
				renderToolChips();
				markDirty();
			});
			chip.appendChild(rm);
			wrap.appendChild(chip);
		});
	}

	function loadTools() {
		if (toolState.loaded) return Promise.resolve(toolState);
		return fetch("/api/tools")
			.then(function (r) {
				if (!r.ok) throw new Error("tool list unavailable (" + r.status + ")");
				return r.json();
			})
			.then(function (list) {
				toolState.available = list.map(function (t) {
					return t.name;
				});
				toolState.meta = list;
				toolState.loaded = true;
				return toolState;
			})
			.catch(function (err) {
				toolState.error = err.message || String(err);
				toolState.loaded = true;
				toolState.available = null;
				return toolState;
			});
	}

	function initTools() {
		var store = $("[data-tools-store]");
		if (!store || store.dataset.bound) return;
		store.dataset.bound = "1";

		renderToolChips();
		loadTools().then(renderToolChips);

		var dialog = $("#tool-picker");
		var open = $("[data-tool-open]");
		if (!dialog || !open) return;

		var listEl = $("[data-tool-list]", dialog);
		var searchEl = $("[data-tool-search]", dialog);
		var statusEl = $("[data-tool-status]", dialog);

		function paint() {
			var picked = selectedTools();
			var q = (searchEl.value || "").trim().toLowerCase();
			listEl.innerHTML = "";

			if (toolState.error || !toolState.meta) {
				statusEl.textContent =
					toolState.error || "No MCP server is connected, so no tools can be listed.";
				return;
			}

			// A tool that no connected server offers any more is still selected;
			// show it at the head of the list so it can be seen and removed here.
			var stale = picked
				.filter(function (name) {
					return toolState.available.indexOf(name) === -1;
				})
				.map(function (name) {
					return { name: name, description: "", stale: true };
				});

			var rows = stale.concat(toolState.meta).filter(function (t) {
				return (
					!q ||
					t.name.toLowerCase().indexOf(q) !== -1 ||
					(t.description || "").toLowerCase().indexOf(q) !== -1
				);
			});

			function status(count) {
				var line =
					rows.length + " of " + (toolState.meta.length + stale.length) +
					" tools · " + (count ? count + " selected" : "none selected, so all are allowed");
				if (stale.length) {
					line += " · " + stale.length + " no longer offered by any connected server";
				}
				return line;
			}

			statusEl.textContent = status(picked.length);

			rows.forEach(function (t) {
				var row = doc.createElement("label");
				row.className = "pick-row";
				var box = doc.createElement("input");
				box.type = "checkbox";
				box.value = t.name;
				box.checked = picked.indexOf(t.name) !== -1;
				box.addEventListener("change", function () {
					var current = selectedTools();
					if (box.checked) {
						if (current.indexOf(t.name) === -1) current.push(t.name);
					} else {
						current = current.filter(function (x) {
							return x !== t.name;
						});
					}
					store.value = current.join(", ");
					renderToolChips();
					markDirty();
					statusEl.textContent = status(current.length);
				});
				var text = doc.createElement("span");
				text.innerHTML =
					'<span class="pick-name"' +
					(t.stale ? ' data-stale="true"' : "") +
					">" +
					escapeHTML(t.name) +
					"</span>" +
					(t.stale
						? '<span class="pick-desc">No connected MCP server offers this tool. It is kept but never called.</span>'
						: t.description
						? '<span class="pick-desc">' + escapeHTML(t.description) + "</span>"
						: "");
				row.appendChild(box);
				row.appendChild(text);
				listEl.appendChild(row);
			});
		}

		open.addEventListener("click", function () {
			statusEl.textContent = "Loading tools…";
			listEl.innerHTML = "";
			dialog.showModal();
			searchEl.focus();
			loadTools().then(paint);
		});
		searchEl.addEventListener("input", paint);
		$$("[data-dialog-close]", dialog).forEach(function (btn) {
			btn.addEventListener("click", function () {
				dialog.close();
			});
		});
		var clear = $("[data-tool-clear]", dialog);
		if (clear) {
			clear.addEventListener("click", function () {
				store.value = "";
				renderToolChips();
				markDirty();
				paint();
			});
		}
	}

	/* --- Editor: import labels from a Grafana alert rule ---------------- */

	function initGrafanaImport() {
		var open = $("[data-grafana-open]");
		var dialog = $("#grafana-import");
		if (!open || !dialog || open.dataset.bound) return;
		open.dataset.bound = "1";

		var listEl = $("[data-grafana-list]", dialog);
		var searchEl = $("[data-grafana-search]", dialog);
		var statusEl = $("[data-grafana-status]", dialog);
		var rules = [];

		function labelsOf(rule) {
			var labels = rule.labels || {};
			var pairs = [{ key: "alertname", value: rule.title }];
			Object.keys(labels).forEach(function (k) {
				if (k !== "alertname") pairs.push({ key: k, value: labels[k] });
			});
			return pairs;
		}

		function apply(rule) {
			var list = $("[data-pairs]");
			if (!list) return;
			list.innerHTML = "";
			labelsOf(rule).forEach(function (p) {
				list.appendChild(pairRow(p.key, p.value));
			});
			syncPairs();
			markDirty();
			dialog.close();
		}

		function paint() {
			var q = (searchEl.value || "").trim().toLowerCase();
			listEl.innerHTML = "";
			var rows = rules.filter(function (rule) {
				if (!q) return true;
				if ((rule.title || "").toLowerCase().indexOf(q) !== -1) return true;
				if ((rule.rule_group || "").toLowerCase().indexOf(q) !== -1) return true;
				var labels = rule.labels || {};
				return Object.keys(labels).some(function (k) {
					return (k + "=" + labels[k]).toLowerCase().indexOf(q) !== -1;
				});
			});

			statusEl.textContent = rows.length
				? rows.length + " of " + rules.length + " rules"
				: "No rule matches that search.";

			rows.forEach(function (rule) {
				var row = doc.createElement("button");
				row.type = "button";
				row.className = "pick-row";
				var chips = labelsOf(rule)
					.map(function (p) {
						return (
							'<span class="chip"><span class="chip-key">' +
							escapeHTML(p.key) +
							"</span>" +
							escapeHTML(p.value) +
							"</span>"
						);
					})
					.join("");
				row.innerHTML =
					ICON_CHEVRON +
					'<span><span class="pick-name">' +
					escapeHTML(rule.title || "(untitled rule)") +
					"</span>" +
					(rule.rule_group
						? '<span class="pick-desc">' + escapeHTML(rule.rule_group) + "</span>"
						: "") +
					'<span class="chips">' +
					chips +
					"</span></span>";
				row.addEventListener("click", function () {
					apply(rule);
				});
				listEl.appendChild(row);
			});
		}

		open.addEventListener("click", function () {
			dialog.showModal();
			searchEl.value = "";
			listEl.innerHTML = "";
			statusEl.textContent = "Loading alert rules from Grafana…";
			fetch("/api/grafana/alerts")
				.then(function (r) {
					if (!r.ok) throw new Error("Grafana returned " + r.status);
					return r.json();
				})
				.then(function (data) {
					rules = Array.isArray(data) ? data : [];
					if (!rules.length) {
						statusEl.textContent = "Grafana returned no alert rules.";
						return;
					}
					paint();
				})
				.catch(function (err) {
					statusEl.textContent =
						"Could not reach Grafana through MCP (" +
						(err.message || err) +
						"). Add the labels by hand instead.";
				});
		});

		searchEl.addEventListener("input", paint);
		$$("[data-dialog-close]", dialog).forEach(function (btn) {
			btn.addEventListener("click", function () {
				dialog.close();
			});
		});
	}

	/* --- Editor: timeout presets and unsaved state ---------------------- */

	function initPresets() {
		$$("[data-preset]").forEach(function (btn) {
			if (btn.dataset.bound) return;
			btn.dataset.bound = "1";
			btn.addEventListener("click", function () {
				var target = doc.getElementById(btn.dataset.presetTarget);
				if (!target) return;
				target.value = btn.dataset.preset;
				target.dispatchEvent(new Event("input", { bubbles: true }));
				markDirty();
			});
		});
	}

	var dirty = false;

	function markDirty() {
		dirty = true;
		var state = $("[data-save-state]");
		if (state && state.getAttribute("data-state") !== "dirty") {
			state.setAttribute("data-state", "dirty");
			state.textContent = "Unsaved changes";
		}
	}

	function initDirtyGuard() {
		var form = $("[data-guard-form]");
		if (!form || form.dataset.bound) return;
		form.dataset.bound = "1";
		dirty = false;
		form.addEventListener("input", markDirty);
		form.addEventListener("change", markDirty);
		form.addEventListener("submit", function () {
			dirty = false;
		});
		$$("[data-leave]").forEach(function (a) {
			a.addEventListener("click", function () {
				dirty = false;
			});
		});
	}

	// Registered once: the handler looks up the current form at unload time.
	function initLeaveGuard() {
		window.addEventListener("beforeunload", function (e) {
			if (!dirty || !$("[data-guard-form]")) return;
			e.preventDefault();
			e.returnValue = "";
		});
	}

	/* --- Prompts: tabs -------------------------------------------------- */

	function initTabs() {
		var bar = $("[data-tabs]");
		var list = $(".stage-list");
		if (!bar || !list) return;
		var tabs = $$("[data-tab]", bar);
		if (!tabs.length) return;
		list.setAttribute("data-tabbed", "");

		function select(id, focus) {
			if (!id || !doc.getElementById(id)) id = tabs[0].dataset.tab;
			tabs.forEach(function (t) {
				var on = t.dataset.tab === id;
				t.setAttribute("aria-selected", on ? "true" : "false");
				t.tabIndex = on ? 0 : -1;
				if (on && focus) t.focus();
			});
			$$("[data-stage]", list).forEach(function (st) {
				if (st.id === id) st.setAttribute("data-active", "");
				else st.removeAttribute("data-active");
			});
			bar.dataset.current = id;
		}

		if (!bar.dataset.bound) {
			bar.dataset.bound = "1";
			bar.addEventListener("click", function (e) {
				var t = e.target.closest("[data-tab]");
				if (!t) return;
				e.preventDefault();
				select(t.dataset.tab);
				try {
					history.replaceState(null, "", "#" + t.dataset.tab);
				} catch (err) {}
			});
			bar.addEventListener("keydown", function (e) {
				if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return;
				e.preventDefault();
				var i = tabs.findIndex(function (t) {
					return t.dataset.tab === bar.dataset.current;
				});
				i = (i + (e.key === "ArrowRight" ? 1 : -1) + tabs.length) % tabs.length;
				select(tabs[i].dataset.tab, true);
			});
		}
		// Re-applied after htmx swaps a stage, which arrives without data-active.
		select(bar.dataset.current || location.hash.slice(1));
	}

	/* --- Prompts: unsaved / saved state --------------------------------- */

	function initPrompts(root) {
		$$("[data-stage]", root).forEach(function (stage) {
			if (stage.dataset.stageBound) return;
			stage.dataset.stageBound = "1";

			var area = $("textarea", stage);
			var state = $("[data-stage-state]", stage);
			var revert = $("[data-stage-revert]", stage);
			var save = $("[data-stage-save]", stage);
			var original = area ? area.value : "";

			function setState(next) {
				if (!state) return;
				state.setAttribute("data-state", next);
				state.textContent =
					next === "dirty" ? "Unsaved changes" : next === "saved" ? "Saved" : "Up to date";
				var tab = $('[data-tab="' + stage.id + '"]');
				if (tab) tab.setAttribute("data-state", next);
				if (save) save.disabled = next !== "dirty";
				if (revert) revert.disabled = next !== "dirty";
			}

			if (area) {
				setState(stage.dataset.saved === "true" ? "saved" : "clean");
				area.addEventListener("input", function () {
					setState(area.value === original ? "clean" : "dirty");
				});
			}

			if (revert) {
				revert.addEventListener("click", function () {
					area.value = original;
					resize(area);
					setState("clean");
					area.focus();
				});
			}
		});
	}

	/* --- Boot ----------------------------------------------------------- */

	// mount is idempotent: htmx replaces whole page bodies and fragments, so
	// everything here has to survive being called again against fresh DOM.
	function mount(root) {
		initTheme();
		initFilter();
		initProbe();
		initPairs();
		initTools();
		initGrafanaImport();
		initPresets();
		initDirtyGuard();
		initAutoResize(root || doc);
		initPrompts(root || doc);
		initTabs();
	}

	// After a reorder, keep keyboard focus on the arrow that was pressed.
	var pendingMove = null;

	doc.addEventListener("DOMContentLoaded", function () {
		initDeleteConfirm();
		initSlashKey();
		initLeaveGuard();
		mount(doc);
	});

	doc.body.addEventListener("htmx:beforeRequest", function (e) {
		var btn = e.target.closest && e.target.closest("[data-move]");
		pendingMove = btn ? { row: btn.closest(".scn-row").id, dir: btn.dataset.move } : null;
	});

	// htmx replaced a prompt stage after a save. The Save button is disabled
	// again, so focus goes back to the field being edited. This waits for
	// afterSettle: focusing on afterSwap is undone by htmx settling the swap.
	doc.body.addEventListener("htmx:afterSettle", function () {
		var stage = $('[data-stage][data-saved="true"]:not([data-refocused])');
		if (!stage) return;
		stage.setAttribute("data-refocused", "");
		var area = $("textarea", stage);
		if (!area) return;
		area.focus();
		var end = area.value.length;
		try {
			area.setSelectionRange(end, end);
		} catch (e) {}
	});

	doc.body.addEventListener("htmx:afterSwap", function () {
		mount(doc);
		if (pendingMove) {
			var row = doc.getElementById(pendingMove.row);
			var btn = row && $('[data-move="' + pendingMove.dir + '"]', row);
			if (btn && btn.disabled) btn = $("[data-move]:not([disabled])", row);
			if (btn) btn.focus();
			pendingMove = null;
		}
	});

	if (doc.fonts && doc.fonts.ready) {
		doc.fonts.ready.then(function () {
			initAutoResize(doc);
		});
	}
})();
