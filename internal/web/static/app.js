(() => {
  const supportedLanguages = new Set([
    "ru", "en", "uk", "be", "pl", "de", "es", "fr", "it", "pt",
    "tr", "ar", "id", "vi", "ja", "ko", "zh", "nl", "cs", "sv",
  ]);
  const language = () => {
    const raw = (document.documentElement.lang || "ru").toLowerCase().trim();
    const base = raw.split(/[-_]/, 1)[0];
    return supportedLanguages.has(base) ? base : "ru";
  };
  const copy = {
    ru: {
      loginOpenTitle: "Открой студию в Telegram",
      loginOpenMessage: "Войди через Mini App бота — твои данные будут под рукой.",
      loginExpired: "Вход устарел. Закрой Mini App и открой его снова.",
      loginFailed: "Не удалось войти. Попробуй ещё раз.",
      connectFailed: "Не удалось подключиться",
      openFailed: "Не удалось открыть студию.",
      sessionExpired: "Сессия закончилась. Закрой приложение и открой его снова.",
      forbidden: "У тебя нет доступа к этому действию.",
      fileTooLarge: "Файл слишком большой. CSV — до 2 МБ, музыка — до 20 МБ.",
      actionFailed: "Не удалось выполнить действие. Попробуй ещё раз.",
      noConnection: "Нет связи с сервером. Попробуй ещё раз.",
      chooseFile: "Нажми, чтобы выбрать файл",
      allReady: "Все {{videos}} ролика готовы",
      progressLabel: "Ролик {{video}} из {{videos}} · реклама {{within}} из 2",
      progressDone: "Все рекламные просмотры засчитаны — запускаем создание видео.",
      progressCurrent: "На каждый ролик — два полных просмотра. Сейчас {{count}} общего прогресса.",
      cooldownReady: "Можно открыть следующую рекламу.",
      cooldown: "Следующая реклама будет доступна через {{seconds}} сек…",
      openInside: "Открой Mini App внутри Telegram, чтобы смотреть рекламу.",
      notSetup: "Реклама ещё не настроена. Выбери оплату Stars.",
      openSessionFailed: "Не удалось открыть рекламную сессию",
      promoApplied: "Промокод применён. Открываем очередь видео…",
      sessionMissing: "Рекламная сессия не создана",
      confirming: "Подтверждаем предыдущий просмотр…",
      loadingAd: "Загружаем рекламу…",
      throttle: "AdsGram просит немного подождать между рекламами.",
      adsNotLoaded: "AdsGram не загрузился",
      showFailed: "Рекламу не удалось показать полностью. Попробуй ещё раз или выбери Stars.",
      notComplete: "Просмотр не завершён",
      verifying: "Проверяем просмотр {{step}} из {{total}}…",
      serverNoConfirm: "Сервер не подтвердил просмотр",
      doneLabel: "Все просмотры завершены",
      doneStatus: "Готово. Открываем очередь видео…",
      startWatching: "Начать просмотр",
      continueWatching: "Продолжить просмотр",
      continueLabel: "Продолжить: реклама {{step}}",
      adsError: "Рекламу не удалось показать. Попробуй Stars.",
      demoWatching: "Просмотр идёт…",
      demoStatus: "Тестовый просмотр {{current}} из {{total}}…",
      demoPair: "Пара для ролика завершена. Можно перейти к следующему ролику.",
      demoFirst: "Первая реклама для этого ролика завершена. Можно открыть вторую.",
      demoPayment: "Проверяем тестовый платёж…",
      demoPaymentFailed: "Не удалось завершить тестовый платёж",
      demoDone: "Тест рекламы завершён, но заказ не удалось оформить. Попробуй ещё раз.",
    },
    en: {
      loginOpenTitle: "Open the studio in Telegram",
      loginOpenMessage: "Sign in through the bot Mini App to keep your data at hand.",
      loginExpired: "The sign-in expired. Close the Mini App and open it again.",
      loginFailed: "Could not sign in. Try again.",
      connectFailed: "Could not connect",
      openFailed: "Could not open the studio.",
      sessionExpired: "Your session ended. Close the app and open it again.",
      forbidden: "You do not have access to this action.",
      fileTooLarge: "File is too large. CSV up to 2 MB, music up to 20 MB.",
      actionFailed: "Could not complete the action. Try again.",
      noConnection: "No connection to the server. Try again.",
      chooseFile: "Click to choose a file",
      allReady: "All {{videos}} videos are ready",
      progressLabel: "Video {{video}} of {{videos}} · ad {{within}} of 2",
      progressDone: "All ad views counted — starting video creation.",
      progressCurrent: "Each video needs two complete views. Overall progress: {{count}}.",
      cooldownReady: "You can open the next ad.",
      cooldown: "The next ad will be available in {{seconds}} sec…",
      openInside: "Open the Mini App inside Telegram to watch an ad.",
      notSetup: "Ads are not configured yet. Choose Stars payment.",
      openSessionFailed: "Could not open the ad session",
      promoApplied: "Promo code applied. Opening the video queue…",
      sessionMissing: "Ad session was not created",
      confirming: "Confirming the previous view…",
      loadingAd: "Loading ad…",
      throttle: "AdsGram asks you to wait briefly between ads.",
      adsNotLoaded: "AdsGram did not load",
      showFailed: "The ad was not watched to completion. Try again or choose Stars.",
      notComplete: "View was not completed",
      verifying: "Checking view {{step}} of {{total}}…",
      serverNoConfirm: "The server did not confirm the view",
      doneLabel: "All views completed",
      doneStatus: "Done. Opening the video queue…",
      startWatching: "Start watching",
      continueWatching: "Continue watching",
      continueLabel: "Continue: ad {{step}}",
      adsError: "The ad could not be shown. Try Stars.",
      demoWatching: "Watching…",
      demoStatus: "Test view {{current}} of {{total}}…",
      demoPair: "The pair for this video is complete. Move to the next video.",
      demoFirst: "The first ad for this video is complete. Open the second one.",
      demoPayment: "Checking test payment…",
      demoPaymentFailed: "Could not complete the test payment",
      demoDone: "The ad test ended, but the order could not be placed. Try again.",
    },
  };
  function tr(key, values = {}) {
    const active = copy[language()] || copy.en || copy.ru;
    const text = active[key] || copy.en[key] || copy.ru[key] || key;
    return text.replace(/\{\{(\w+)\}\}/g, (_, name) => String(values[name] ?? ""));
  }
  const tg = window.Telegram?.WebApp;
  const title = () => document.getElementById("login-title");
  const message = () => document.getElementById("login-message");
  let authenticating = false;
  let focusPreviewAfterUpdate = false;
  function report(text) {
    const error = document.getElementById("studio-error");
    if (error) {
      error.textContent = text;
      error.hidden = false;
    } else if (message()) {
      message().textContent = text;
    }
  }
  async function login(dev = false) {
    if (authenticating) return;
    if (!dev && !tg?.initData) {
      if (title()) title().textContent = tr("loginOpenTitle");
      if (message())
        message().textContent =
          tr("loginOpenMessage");
      return;
    }
    authenticating = true;
    const button = document.getElementById("dev-login");
    if (button) button.disabled = true;
    try {
      const response = await fetch(dev ? "/auth/dev" : "/auth/telegram", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(dev ? {} : { init_data: tg.initData }),
      });
      if (!response.ok) {
        throw new Error(
          response.status === 401
            ? tr("loginExpired")
            : tr("loginFailed"),
        );
      }
      const screens = ["/studio", "/library", "/cart", "/status", "/admin"];
      const path = screens.includes(window.location.pathname)
        ? window.location.pathname + window.location.search
        : "/studio";
      await htmx.ajax("GET", path, {
        target: "#studio",
        swap: "outerHTML",
      });
      document.body.dataset.authenticated = "true";
    } catch (error) {
      if (title()) title().textContent = tr("connectFailed");
      report(error.message || tr("openFailed"));
      const retry = document.getElementById("retry-login");
      if (retry) retry.hidden = false;
    } finally {
      authenticating = false;
      if (button) button.disabled = false;
    }
  }
  if (tg) {
    tg.ready();
    tg.expand();
    if (tg.isVersionAtLeast?.("6.1")) {
      tg.setHeaderColor("#101215");
      tg.setBackgroundColor("#101215");
    }
    const applySafeArea = () => {
      const top =
        (tg.safeAreaInset?.top || 0) + (tg.contentSafeAreaInset?.top || 0);
      const bottom =
        (tg.safeAreaInset?.bottom || 0) +
        (tg.contentSafeAreaInset?.bottom || 0);
      document.documentElement.style.setProperty("--safe-top", `${top}px`);
      document.documentElement.style.setProperty(
        "--safe-bottom",
        `${bottom}px`,
      );
    };
    applySafeArea();
    tg.onEvent("safeAreaChanged", applySafeArea);
    tg.onEvent("contentSafeAreaChanged", applySafeArea);
  }
  document
    .getElementById("dev-login")
    ?.addEventListener("click", () => login(true));
  document
    .getElementById("retry-login")
    ?.addEventListener("click", () =>
      login(!!document.getElementById("dev-login") && !tg?.initData),
    );
  document.body.addEventListener("htmx:responseError", (event) => {
    if (event.detail.xhr.status === 401)
      report(tr("sessionExpired"));
    else if (event.detail.xhr.status === 403)
      report(tr("forbidden"));
    else if (event.detail.xhr.status === 413)
      report(tr("fileTooLarge"));
    else report(tr("actionFailed"));
  });
  document.body.addEventListener("htmx:sendError", () =>
    report(tr("noConnection")),
  );
  document.body.addEventListener("change", (event) => {
    if (event.target.matches('input[name="theme"]')) {
      focusPreviewAfterUpdate = true;
    }
    const choice = event.target.closest?.("[data-payment-choice]");
    if (choice) {
      const value = choice.value;
      document.querySelectorAll("[data-payment-choice-card]").forEach((card) => {
        card.classList.toggle("is-selected", card.querySelector("[data-payment-choice]") === choice);
      });
      document.querySelectorAll("[data-payment-panel]").forEach((panel) => {
        panel.hidden = panel.dataset.paymentPanel !== value;
      });
      return;
    }
    if (event.target.matches('input[type="file"]')) {
      const label = event.target.closest(".file-field")?.querySelector("[data-file-label]");
      if (label) label.textContent = event.target.files[0]?.name || tr("chooseFile");
    }
  });

  let adsController = null;
  let adsControllerBlock = "";

  function adsStorageKey(orderID) {
    return `contgen:ads:${orderID}`;
  }

  function readAdsState(orderID) {
    try {
      return JSON.parse(sessionStorage.getItem(adsStorageKey(orderID)) || "null");
    } catch (_) {
      return null;
    }
  }

  function writeAdsState(orderID, state) {
    try {
      sessionStorage.setItem(adsStorageKey(orderID), JSON.stringify(state));
    } catch (_) {}
  }

  function clearAdsState(orderID) {
    try {
      sessionStorage.removeItem(adsStorageKey(orderID));
    } catch (_) {}
  }

  function adCountLabel(value, total) {
    return `${String(value).padStart(2, "0")} / ${String(total).padStart(2, "0")}`;
  }

  function updateAdProgress(panel, completed, total) {
    if (!panel) return;
    total = Math.max(1, Number(total) || 2);
    completed = Math.max(0, Math.min(total, Number(completed) || 0));
    const videos = Math.max(1, Number(panel.dataset.adVideos) || Math.ceil(total / 2));
    const video = Math.min(videos, Math.floor(completed / 2) + 1);
    const within = completed >= total ? 2 : (completed % 2) + 1;
    const percent = Math.round((completed / total) * 100);
    const label = panel.querySelector("[data-ad-progress-label]");
    const fill = panel.querySelector("[data-ad-progress-fill]");
    const percentNode = panel.querySelector("[data-ad-progress-percent]");
    const currentVideo = panel.querySelector("[data-ad-current-video]");
    const summaryCount = panel.querySelector("[data-ad-summary-count]");
    const actionCount = panel.querySelector("[data-ad-action-count]");
    if (label) label.textContent = completed >= total
      ? tr("allReady", { videos })
      : tr("progressLabel", { video, videos, within });
    if (fill) fill.style.width = `${percent}%`;
    if (percentNode) percentNode.textContent = `${percent}%`;
    if (summaryCount) summaryCount.textContent = adCountLabel(completed, total);
    if (actionCount) actionCount.textContent = completed >= total
      ? `${adCountLabel(total, total)} ✓`
      : `${adCountLabel(completed + 1, total)} ↗`;
    if (currentVideo) currentVideo.textContent = completed >= total
      ? tr("progressDone")
      : tr("progressCurrent", { count: adCountLabel(completed + 1, total) });
    panel.querySelectorAll("[data-ad-step]").forEach((step) => {
      const number = Number(step.dataset.adStep || 0);
      step.classList.toggle("is-done", completed >= total || number < within);
      step.classList.toggle("is-current", completed < total && number === within);
      const mark = step.querySelector("b");
      if (mark) mark.textContent = completed >= total || number < within ? "✓" : "○";
    });
  }

  async function cancelAdsSession(session) {
    if (!session) return;
    try {
      await fetch("/cart/ads/cancel", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ session }),
        keepalive: true,
      });
    } catch (_) {}
  }

  function startAdsCooldown(button, status, milliseconds) {
    const until = Date.now() + milliseconds;
    button.dataset.adsCooldownUntil = String(until);
    button.disabled = true;
    const panel = button.closest(".ad-payment");
    const total = Number(button.dataset.adsTotal || panel?.dataset.adTotal || 2);
    const completed = Number(button.dataset.adCurrent || 0);
    updateAdProgress(panel, completed, total);
    const tick = () => {
      const remaining = Math.max(0, Number(button.dataset.adsCooldownUntil || 0) - Date.now());
      if (remaining <= 0) {
        button.disabled = false;
        status.textContent = tr("cooldownReady");
        return;
      }
      status.textContent = tr("cooldown", { seconds: Math.ceil(remaining / 1000) });
      window.setTimeout(tick, 500);
    };
    tick();
  }

  async function runAdsGram(button, status) {
    const blockId = document.body.dataset.adsgramBlockId;
    const debug = document.body.dataset.adsgramDebug === "true";
    const form = button.closest("form");
    const panel = button.closest(".ad-payment");
    const orderID = form?.querySelector('input[name="order_id"]')?.value || "";
    const promoCode = form?.querySelector('input[name="code"]')?.value || "";
    if (!tg?.initData) {
      status.textContent = tr("openInside");
      status.hidden = false;
      return;
    }
    if (!blockId || !window.Adsgram?.init) {
      status.textContent = tr("notSetup");
      status.hidden = false;
      return;
    }
    const label = button.querySelector("[data-ad-label]");
    const count = button.querySelector("[data-ad-action-count]");
    const stored = readAdsState(orderID);
    let session = button.dataset.adsSession || stored?.session || "";
    let current = Number(button.dataset.adCurrent || stored?.completed || "0");
    let total = Number(button.dataset.adsTotal || stored?.total || panel?.dataset.adTotal || "2");
    total = Math.max(1, total);
    button.dataset.adsTotal = String(total);
    updateAdProgress(panel, current, total);
    if (session) {
      try {
        const stateResponse = await fetch(`/cart/ads/status?session=${encodeURIComponent(session)}`, { credentials: "same-origin" });
        if (stateResponse.ok) {
          const state = await stateResponse.json();
          current = Number(state.completed ?? current);
          total = Math.max(1, Number(state.total ?? total));
          button.dataset.adsTotal = String(total);
          if (state.done) {
            updateAdProgress(panel, total, total);
            clearAdsState(orderID);
            window.location.assign("/status");
            return;
          }
          if (state.finalizing) {
            status.hidden = false;
            status.textContent = tr("confirming");
            window.setTimeout(() => runAdsGram(button, status), 1000);
            return;
          }
          writeAdsState(orderID, { session, completed: current, total });
        } else if (stateResponse.status === 410) {
          clearAdsState(orderID);
          session = "";
          current = 0;
        }
      } catch (_) {}
    }
    try {
      if (!session) {
        const start = await fetch("/cart/ads/start", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: new URLSearchParams({ order_id: orderID, code: promoCode }),
        });
        if (!start.ok) throw new Error(tr("openSessionFailed"));
        const data = await start.json();
        if (data.promo || data.done) {
          clearAdsState(orderID);
          status.textContent = tr("promoApplied");
          window.setTimeout(() => { window.location.assign("/status"); }, 450);
          return;
        }
        session = data.token;
        if (!session) throw new Error(tr("sessionMissing"));
        button.dataset.adsSession = session;
        current = Number(data.completed ?? 0);
        total = Math.max(1, Number(data.total ?? total));
        button.dataset.adsTotal = String(total);
        writeAdsState(orderID, { session, completed: current, total });
      }
      button.dataset.adCurrent = String(current);
      updateAdProgress(panel, current, total);
      if (current >= total) {
        window.location.assign("/status");
        return;
      }
      button.disabled = true;
      status.hidden = false;
      status.textContent = tr("loadingAd");
      if (!adsController || adsControllerBlock !== blockId) {
        adsController = window.Adsgram.init({ blockId, debug, debugConsole: debug, debugBannerType: debug ? "RewardedVideo" : undefined });
        adsControllerBlock = blockId;
        adsController.addEventListener?.("onNonStopShow", () => {
          status.textContent = tr("throttle");
        });
      }
      const controller = adsController;
      if (!controller?.show) throw new Error(tr("adsNotLoaded"));
      let result;
      try {
        result = await controller.show();
      } catch (_) {
        throw new Error(tr("showFailed"));
      }
      if (!result || result.done !== true) throw new Error(tr("notComplete"));
      const step = current + 1;
      status.textContent = tr("verifying", { step, total });
      const complete = await fetch("/cart/ads/complete", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ session, step: String(step) }),
      });
      if (!complete.ok) throw new Error(tr("serverNoConfirm"));
      const data = await complete.json();
      current = Number(data.completed ?? step);
      total = Math.max(1, Number(data.total ?? total));
      button.dataset.adCurrent = String(current);
      button.dataset.adsTotal = String(total);
      updateAdProgress(panel, current, total);
      writeAdsState(orderID, { session, completed: current, total });
      if (data.done) {
        button.classList.add("is-complete");
        if (label) label.textContent = tr("doneLabel");
        if (count) count.textContent = `${adCountLabel(total, total)} ✓`;
        status.textContent = tr("doneStatus");
        clearAdsState(orderID);
        window.setTimeout(() => { window.location.assign("/status"); }, 450);
      } else {
        startAdsCooldown(button, status, 5000);
        if (label) label.textContent = tr("continueLabel", { step: step + 1 });
        if (count) count.textContent = `${adCountLabel(step + 1, total)} ↗`;
      }
    } catch (error) {
      const serverConfirmationError = error?.message === tr("serverNoConfirm") || error?.message?.includes("confirm");
      if (!serverConfirmationError && current <= 0) {
        await cancelAdsSession(session);
        clearAdsState(orderID);
        button.dataset.adsSession = "";
        button.dataset.adCurrent = "0";
        updateAdProgress(panel, 0, total);
        if (label) label.textContent = tr("startWatching");
        if (count) count.textContent = `${adCountLabel(1, total)} ↗`;
      } else if (!serverConfirmationError) {
        writeAdsState(orderID, { session, completed: current, total });
        updateAdProgress(panel, current, total);
        if (label) label.textContent = tr("continueLabel", { step: current + 1 });
        if (count) count.textContent = `${adCountLabel(current + 1, total)} ↗`;
      }
      button.disabled = false;
      status.hidden = false;
      status.textContent = error?.message || tr("adsError");
    }
  }

  document.body.addEventListener("click", (event) => {
    const button = event.target.closest("[data-ad-start]");
    if (!button) return;
    const status = button.closest(".ad-payment")?.querySelector("[data-ad-status]");
    if (!status) return;
    if (document.body.dataset.localDemo === "true") {
      if (button.dataset.demoRunning === "true" || button.classList.contains("is-complete")) return;
      const panel = button.closest(".ad-payment");
      const steps = [...panel.querySelectorAll("[data-ad-step]")];
      const label = button.querySelector("[data-ad-label]");
      const count = button.querySelector("[data-ad-action-count]");
      const total = Math.max(1, Number(button.dataset.adsTotal || panel?.dataset.adTotal || "2"));
      let current = Number(button.dataset.adCurrent || "0");
      button.dataset.adsTotal = String(total);
      updateAdProgress(panel, current, total);
      button.dataset.demoRunning = "true";
      const playNext = () => {
        const step = steps[current % steps.length];
        if (!step) return;
        button.disabled = true;
        step.classList.add("is-watching");
        if (label) label.textContent = tr("demoWatching");
        if (count) count.textContent = adCountLabel(current + 1, total);
        updateAdProgress(panel, current, total);
        status.textContent = tr("demoStatus", { current: current + 1, total });
        status.hidden = false;
        window.setTimeout(() => {
          step.classList.remove("is-watching");
          step.classList.add("is-done");
          const mark = step.querySelector("b");
          if (mark) mark.textContent = "✓";
          current += 1;
          button.dataset.adCurrent = String(current);
          updateAdProgress(panel, current, total);
          if (current < total) {
            button.disabled = false;
            button.dataset.demoRunning = "false";
            if (label) label.textContent = tr("continueWatching");
            if (count) count.textContent = `${adCountLabel(current + 1, total)} ↗`;
            status.textContent = current % 2 === 0
              ? tr("demoPair")
              : tr("demoFirst");
            return;
          }
          button.classList.add("is-complete");
          if (label) label.textContent = tr("doneLabel");
          if (count) count.textContent = `${adCountLabel(total, total)} ✓`;
          status.textContent = tr("demoPayment");
          const form = button.closest("form");
          const orderID = form?.querySelector('input[name="order_id"]')?.value;
          fetch("/cart/ads/complete", {
            method: "POST",
            credentials: "same-origin",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: new URLSearchParams({ order_id: orderID || "" }),
          }).then((response) => {
            if (!response.ok) throw new Error(tr("demoPaymentFailed"));
            status.textContent = tr("doneStatus");
            window.setTimeout(() => { window.location.assign("/status"); }, 450);
          }).catch(() => {
            button.disabled = false;
            button.classList.remove("is-complete");
            button.dataset.demoRunning = "false";
            button.dataset.adCurrent = "0";
            status.textContent = tr("demoDone");
          });
        }, 900);
      };
      playNext();
      return;
    }
    runAdsGram(button, status);
  });
  document.body.addEventListener("htmx:xhr:progress", (event) => {
    const progress = event.target.closest("form")?.querySelector("progress");
    if (progress && event.detail.total > 0) {
      progress.hidden = false;
      progress.value = event.detail.loaded / event.detail.total * 100;
    }
  });
  document.body.addEventListener("htmx:afterRequest", (event) => {
    const progress = event.target.closest("form")?.querySelector("progress");
    if (progress) progress.hidden = true;
  });
  document.body.addEventListener("htmx:afterSettle", () => {
    if (!focusPreviewAfterUpdate) return;
    focusPreviewAfterUpdate = false;
    const preview = document.querySelector(".preview-area");
    preview?.scrollIntoView({ behavior: "smooth", block: "start" });
    if (preview) {
      preview.classList.remove("is-refreshing");
      window.requestAnimationFrame(() => preview.classList.add("is-refreshing"));
      window.setTimeout(() => preview.classList.remove("is-refreshing"), 650);
    }
  });
  // A single playing preview keeps library and rendered-video audio intelligible.
  document.addEventListener("play", (event) => {
    if (!event.target.matches("audio,video")) return;
    document.querySelectorAll("audio,video").forEach((media) => {
      if (media !== event.target) media.pause();
    });
  }, true);
  document.body.addEventListener("htmx:confirm", (event) => {
    const form = event.target.closest("form[data-confirm]");
    if (!form) return;
    event.preventDefault();
    if (window.confirm(form.dataset.confirm)) event.detail.issueRequest(true);
  });
  if (document.body.dataset.authenticated !== "true") login();
})();
