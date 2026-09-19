/* Alert Agent panel behaviour.
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

	/* --- Scenario field: filter ----------------------------------------- */

	function initFilter() {
		var input = $("[data-filter-input]");
		var field = $("[data-field]");
		if (!input || !field) return;
		if (input.dataset.bound) return;
		input.dataset.bound = "1";

		var countEl = $("[data-filter-count]");
		var emptyEl = $("[data-filter-empty]");

		function apply() {
			var q = input.value.trim().toLowerCase();
			var shown = 0;
			$$(".path", field).forEach(function (row) {
				var hit = !q || (row.dataset.search || "").indexOf(q) !== -1;
				row.hidden = !hit;
				if (hit) shown++;
			});
			if (countEl) {
				countEl.textContent = q
					? shown + " of " + $$(".path", field).length + " shown"
					: $$(".path", field).length + " total";
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
			// Typing narrows to one path; Enter opens it without touching the mouse.
			if (e.key === "Enter") {
				e.preventDefault();
				var visible = $$(".path", field).filter(function (row) {
					return !row.hidden;
				});
				if (visible.length === 1) {
					var link = $(".path-name a", visible[0]);
					if (link) window.location.href = link.href;
				}
			}
		});

		doc.addEventListener("keydown", function (e) {
			if (e.key !== "/" || e.metaKey || e.ctrlKey || e.altKey) return;
			var tag = (doc.activeElement && doc.activeElement.tagName) || "";
			if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
			e.preventDefault();
			input.focus();
			input.select();
		});

		apply();
	}

	/* --- Scenario field: match probe ------------------------------------ */

	function initProbe() {
		var form = $("[data-probe-form]");
		if (!form || form.dataset.bound) return;
		form.dataset.bound = "1";

		var input = $("[data-probe-input]", form);
		var out = $("[data-probe-result]");
		var field = $("[data-field]");

		function clearLive() {
			$$(".path[data-live]", field).forEach(function (row) {
				row.removeAttribute("data-live");
			});
		}

		function render(outcome, html) {
			if (!out) return;
			out.hidden = false;
			out.setAttribute("data-outcome", outcome);
			out.innerHTML = html;
		}

		form.addEventListener("submit", function (e) {
			e.preventDefault();
			var labels = input.value.trim();
			clearLive();
			if (!labels) {
				render(
					"none",
					"<span class='probe-result-note'>Enter at least one <span class='mono'>key=value</span> label to test.</span>"
				);
				return;
			}
			render("pending", "<span class='probe-result-note'>Matching…</span>");

			fetch("/api/match-probe", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ labels: labels })
			})
				.then(function (res) {
					if (!res.ok) throw new Error("probe failed: " + res.status);
					return res.json();
				})
				.then(function (data) {
					if (data.error) {
						render("error", "<span>" + escapeHTML(data.error) + "</span>");
						return;
					}
					if (!data.matched) {
						render(
							"none",
							"<span><strong>No scenario catches this alert.</strong> <span class='probe-result-note'>Nothing matches these labels and there is no catch-all scenario, so the alert would be dropped.</span></span>"
						);
						return;
					}
					var row = field && field.querySelector('.path[data-id="' + data.id + '"]');
					var hiddenByFilter = false;
					if (row) {
						row.setAttribute("data-live", "true");
						// The filter owns row.hidden. Un-hiding here would leave the
						// count contradicting the field, so say so instead.
						hiddenByFilter = row.hidden;
						if (!hiddenByFilter) {
							row.scrollIntoView({ block: "nearest", behavior: "smooth" });
						}
					}
					var rankEl = row && row.querySelector(".rank-node");
					var rank = rankEl ? rankEl.textContent.trim() : "";
					var total = field ? $$(".path", field).length : 0;
					var where = rank ? ", path " + rank + " of " + total : "";
					var why =
						data.reason === "catch-all"
							? "No labelled scenario matched, so the catch-all takes it."
							: "Matched on " + escapeHTML(data.matchedOn) + ".";
					if (hiddenByFilter) {
						why += " It is hidden by the current filter.";
					}
					render(
						"match",
						"<span><strong>" +
							escapeHTML(data.name) +
							"</strong>" +
							where +
							", catches this alert. <span class='probe-result-note'>" +
							why +
							"</span></span>"
					);
				})
				.catch(function (err) {
					render("error", "<span>" + escapeHTML(String(err.message || err)) + "</span>");
				});
		});
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

	/* --- Scenario field: inline delete confirm --------------------------- */

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
				var row = ask.closest(".path");
				if (!row) return;
				row.setAttribute("data-confirming", "true");
				// The button that was just pressed is now display:none, so move
				// focus deliberately instead of letting it fall to <body>.
				var confirm = $("[data-delete-confirm]", row);
				if (confirm) confirm.focus();
				return;
			}
			var cancel = e.target.closest("[data-delete-cancel]");
			if (cancel) {
				closeConfirm(cancel.closest(".path"), true);
			}
		});

		doc.addEventListener("keydown", function (e) {
			if (e.key !== "Escape") return;
			var row = doc.activeElement && doc.activeElement.closest
				? doc.activeElement.closest('.path[data-confirming="true"]')
				: null;
			if (!row) row = $('.path[data-confirming="true"]');
			if (row) {
				e.stopPropagation();
				closeConfirm(row, true);
			}
		});
	}

	/* --- Editor: margin rail -------------------------------------------- */

	function initRail() {
		var rail = $("[data-rail]");
		if (!rail || rail.dataset.bound) return;
		if (!("IntersectionObserver" in window)) return;
		rail.dataset.bound = "1";
		var links = $$("a", rail);
		var sections = links
			.map(function (a) {
				return doc.getElementById(a.getAttribute("href").slice(1));
			})
			.filter(Boolean);
		if (!sections.length) return;

		function setActive(id) {
			links.forEach(function (a) {
				a.classList.toggle("is-active", a.getAttribute("href") === "#" + id);
			});
		}

		var observer = new IntersectionObserver(
			function (entries) {
				entries.forEach(function (entry) {
					if (entry.isIntersecting) setActive(entry.target.id);
				});
			},
			{ rootMargin: "-80px 0px -60% 0px", threshold: 0 }
		);
		sections.forEach(function (s) {
			observer.observe(s);
		});
		setActive(sections[0].id);
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
				: "catch-all";
		}
	}

	function pairRow(key, value) {
		var row = doc.createElement("div");
		row.className = "pair";
		row.innerHTML =
			'<input type="text" class="code" data-pair-key placeholder="alertname" aria-label="Label name">' +
			'<span class="pair-eq" aria-hidden="true">=</span>' +
			'<input type="text" class="code" data-pair-value placeholder="HighCPU" aria-label="Label value">' +
			'<button type="button" class="icon-button" data-pair-remove aria-label="Remove label">' +
			'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18"/></svg>' +
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
			all.textContent = "every available tool";
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
			rm.textContent = "×";
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
					" tools · " + count + " selected";
				if (stale.length) {
					line +=
						" · " + stale.length + " no longer offered by any connected server";
				}
				return line;
			}

			statusEl.textContent = status(picked.length);

			rows.forEach(function (t) {
				var row = doc.createElement("label");
				row.className = "tool-row";
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
					'<span class="tool-name"' +
					(t.stale ? ' data-stale="true"' : "") +
					">" +
					escapeHTML(t.name) +
					"</span>" +
					(t.stale
						? '<span class="tool-desc">No connected MCP server offers this tool. It is kept but never called.</span>'
						: t.description
						? '<span class="tool-desc">' + escapeHTML(t.description) + "</span>"
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


	/* --- Editor: import labels from a Grafana alert rule ------------------ */

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
				row.className = "tool-row";
				row.style.textAlign = "left";
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
					'<span aria-hidden="true">&rsaquo;</span>' +
					'<span><span class="tool-name">' +
					escapeHTML(rule.title || "(untitled rule)") +
					"</span>" +
					(rule.rule_group
						? '<span class="tool-desc">' + escapeHTML(rule.rule_group) + "</span>"
						: "") +
					'<span class="chips" style="margin-top:4px">' +
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

	/* --- Editor: timeout presets and dirty guard ------------------------- */

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
	}

	function initDirtyGuard() {
		var form = $("[data-guard-form]");
		if (!form || form.dataset.bound) return;
		form.dataset.bound = "1";
		dirty = false;
		form.addEventListener("input", markDirty);
		form.addEventListener("submit", function () {
			dirty = false;
		});
		window.addEventListener("beforeunload", function (e) {
			if (!dirty) return;
			e.preventDefault();
			e.returnValue = "";
		});
		$$("[data-leave]").forEach(function (a) {
			a.addEventListener("click", function () {
				dirty = false;
			});
		});
	}

	/* --- Prompts: dirty / saved state ------------------------------------ */

	function initPrompts(root) {
		$$("[data-stage]", root).forEach(function (stage) {
			if (stage.dataset.stageBound) return;
			stage.dataset.stageBound = "1";

			// htmx replaced this stage's markup. The Save button is disabled again
			// after a save, so focus lands on the field the user was editing.
			if (stage.dataset.saved === "true") {
				var savedArea = $("textarea", stage);
				if (savedArea) {
					savedArea.focus();
					var end = savedArea.value.length;
					try {
						savedArea.setSelectionRange(end, end);
					} catch (e) {}
				}
			}

			var area = $("textarea", stage);
			var state = $("[data-stage-state]", stage);
			var revert = $("[data-stage-revert]", stage);
			var save = $("[data-stage-save]", stage);
			var original = area ? area.value : "";

			function setState(next) {
				if (!state) return;
				state.setAttribute("data-state", next);
				state.textContent =
					next === "dirty" ? "Unsaved changes" : next === "saved" ? "Saved" : "In sync";
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

	/* --- Boot ------------------------------------------------------------ */

	// mount is idempotent: htmx replaces whole page bodies, so everything here
	// has to survive being called again against fresh DOM.
	function mount(root) {
		initTheme();
		initFilter();
		initProbe();
		initRail();
		initPairs();
		initTools();
		initGrafanaImport();
		initPresets();
		initDirtyGuard();
		initAutoResize(root || doc);
		initPrompts(root || doc);
	}

	doc.addEventListener("DOMContentLoaded", function () {
		initDeleteConfirm();
		mount(doc);
	});

	doc.body.addEventListener("htmx:afterSwap", function () {
		mount(doc);
	});

	if (doc.fonts && doc.fonts.ready) {
		doc.fonts.ready.then(function () {
			initAutoResize(doc);
		});
	}
})();
