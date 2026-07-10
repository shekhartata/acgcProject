(() => {
  const $ = (id) => document.getElementById(id);

  let sessionId = null;
  let playing = false;

  const btnStart = $("btnStart");
  const btnNext = $("btnNext");
  const btnPlay = $("btnPlay");
  const btnProbe = $("btnProbe");
  const btnReset = $("btnReset");

  function setError(msg) {
    $("error").textContent = msg || "";
  }

  function setBusy(busy) {
    [btnStart, btnNext, btnPlay, btnProbe, btnReset].forEach((b) => {
      if (!b) return;
      if (busy) b.disabled = true;
    });
    if (!busy) syncButtons();
  }

  function syncButtons() {
    const has = !!sessionId;
    btnStart.disabled = has;
    btnNext.disabled = !has || playing;
    btnPlay.disabled = !has || playing;
    btnProbe.disabled = !has || playing;
    btnReset.disabled = !has || playing;
  }

  function renderChat(el, lines) {
    el.innerHTML = "";
    (lines || []).forEach((m) => {
      const div = document.createElement("div");
      div.className = `bubble ${m.role === "user" ? "user" : "assistant"}`;
      const role = document.createElement("span");
      role.className = "role";
      role.textContent = m.role;
      div.appendChild(role);
      div.appendChild(document.createTextNode(m.content || ""));
      el.appendChild(div);
    });
    el.scrollTop = el.scrollHeight;
  }

  function fmtNaiveStats(s) {
    if (!s) return "—";
    if (s.error) return `error: ${s.error}`;
    return `prompt=${s.prompt_tokens}  cum=${s.cumulative_prompt_tokens}  completion=${s.completion_tokens}  history_msgs=${s.history_msgs}`;
  }

  function fmtAcgcStats(s) {
    if (!s) return "—";
    if (s.error) return `error: ${s.error}`;
    const gc = s.gc_triggered ? `gc=${s.gc_reason || "yes"}` : "gc=no";
    return `compiled=${s.prompt_tokens}  original=${s.original_tokens}  saved=${(s.reduction_pct || 0).toFixed(1)}%  nodes a/c/z=${s.active_nodes}/${s.compressed_nodes}/${s.archived_nodes}  budget=${s.token_budget}  ${gc}`;
  }

  async function api(path, body) {
    const res = await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body || {}),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || res.statusText);
    return data;
  }

  async function start() {
    setError("");
    setBusy(true);
    try {
      const data = await api("/api/demo/start", {});
      sessionId = data.session_id;
      $("subtitle").textContent = data.subtitle || $("subtitle").textContent;
      $("metaBudget").textContent = `budget=${data.budget}`;
      $("metaModel").textContent = `model=${data.model}`;
      renderChat($("naiveChat"), data.naive_transcript || []);
      renderChat($("acgcChat"), data.acgc_transcript || []);
      $("naiveStats").textContent = "—";
      $("acgcStats").textContent = "—";
      $("naivePreview").textContent = "";
      $("acgcPreview").textContent = "";
      $("metaProgress").textContent = `warm 0/${data.warm_user_steps}`;
      $("takeaway").textContent = "Early decisions seeded into both panes. Click Next (or Play), then Probe.";
    } catch (e) {
      setError(e.message);
      sessionId = null;
    } finally {
      setBusy(false);
    }
  }

  function applyPane(resp) {
    if (resp.naive) {
      renderChat($("naiveChat"), resp.naive.transcript);
      if (resp.naive.naive_stats) {
        $("naiveStats").textContent = fmtNaiveStats(resp.naive.naive_stats);
        $("naivePreview").textContent = resp.naive.naive_stats.prompt_preview || "";
      }
    }
    if (resp.acgc) {
      renderChat($("acgcChat"), resp.acgc.transcript);
      if (resp.acgc.acgc_stats) {
        $("acgcStats").textContent = fmtAcgcStats(resp.acgc.acgc_stats);
        $("acgcPreview").textContent = resp.acgc.acgc_stats.prompt_preview || "";
      }
    }
  }

  async function nextOnce() {
    const data = await api("/api/demo/next", { session_id: sessionId });
    applyPane(data);
    const done = data.done ? "done" : `${data.warm_remaining} left`;
    $("metaProgress").textContent = `turn ${data.turn_index} (${done})`;
    if (data.done) {
      $("takeaway").textContent = "Warm turns complete. Run Probe to test early-decision recall.";
    }
    return data;
  }

  async function next() {
    if (!sessionId) return;
    setError("");
    setBusy(true);
    try {
      await nextOnce();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function play() {
    if (!sessionId || playing) return;
    playing = true;
    setError("");
    syncButtons();
    try {
      for (;;) {
        const data = await nextOnce();
        if (data.done) break;
        await new Promise((r) => setTimeout(r, 800));
      }
    } catch (e) {
      setError(e.message);
    } finally {
      playing = false;
      syncButtons();
    }
  }

  async function probe() {
    if (!sessionId) return;
    setError("");
    setBusy(true);
    try {
      const data = await api("/api/demo/probe", { session_id: sessionId });
      $("takeaway").textContent = data.takeaway || "";
      const n = data.naive || {};
      const a = data.acgc || {};
      // Append probe Q/A visually via stats panels
      if (n.stats) {
        $("naiveStats").textContent = fmtNaiveStats(n.stats) + `  needle=${n.hit_needle}`;
        if (n.stats.prompt_preview) $("naivePreview").textContent = n.stats.prompt_preview;
      }
      if (a.stats) {
        $("acgcStats").textContent = fmtAcgcStats(a.stats) + `  needle=${a.hit_needle}`;
        if (a.stats.prompt_preview) $("acgcPreview").textContent = a.stats.prompt_preview;
      }
      // Show probe answers as assistant bubbles
      const nChat = $("naiveChat");
      const aChat = $("acgcChat");
      const add = (el, role, text) => {
        const div = document.createElement("div");
        div.className = `bubble ${role === "user" ? "user" : "assistant"}`;
        const r = document.createElement("span");
        r.className = "role";
        r.textContent = role + (role === "assistant" ? " (probe)" : " (probe)");
        div.appendChild(r);
        div.appendChild(document.createTextNode(text || ""));
        el.appendChild(div);
        el.scrollTop = el.scrollHeight;
      };
      add(nChat, "user", data.question);
      add(nChat, "assistant", n.answer || n.error || "");
      add(aChat, "user", data.question);
      add(aChat, "assistant", a.answer || a.error || "");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function reset() {
    setError("");
    if (sessionId) {
      try {
        await api("/api/demo/reset", { session_id: sessionId });
      } catch (_) {}
    }
    sessionId = null;
    playing = false;
    $("metaBudget").textContent = "budget=—";
    $("metaModel").textContent = "model=—";
    $("metaProgress").textContent = "turns —";
    $("takeaway").textContent = "Start a session, step through warm turns, then run the recall probe.";
    renderChat($("naiveChat"), []);
    renderChat($("acgcChat"), []);
    $("naiveStats").textContent = "—";
    $("acgcStats").textContent = "—";
    $("naivePreview").textContent = "";
    $("acgcPreview").textContent = "";
    syncButtons();
  }

  btnStart.addEventListener("click", start);
  btnNext.addEventListener("click", next);
  btnPlay.addEventListener("click", play);
  btnProbe.addEventListener("click", probe);
  btnReset.addEventListener("click", reset);
  syncButtons();
})();
