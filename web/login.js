"use strict";

if (localStorage.getItem("token")) location.replace("/");

const stepEmail    = document.getElementById("stepEmail");
const stepPassword = document.getElementById("stepPassword");

let confirmedEmail = "";

function showFieldError(textId, wrapperId, msg) {
  document.getElementById(textId).textContent = msg;
  document.getElementById(wrapperId).classList.add("visible");
}

function clearFieldError(wrapperId) {
  document.getElementById(wrapperId).classList.remove("visible");
}

function setInputError(inputId, hasError) {
  document.getElementById(inputId).classList.toggle("error", hasError);
}

function isValidEmail(v) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(v);
}

// ── Шаг 1: Email ─────────────────────────────────────────────────────────────

document.getElementById("formEmail").onsubmit = async (e) => {
  e.preventDefault();
  const email = document.getElementById("inputEmail").value.trim();
  const btn   = document.getElementById("btnNext");

  clearFieldError("errorEmail");
  setInputError("inputEmail", false);

  if (!isValidEmail(email)) {
    showFieldError("errorEmailText", "errorEmail", "Укажите действительный адрес электронной почты.");
    setInputError("inputEmail", true);
    return;
  }

  btn.disabled = true;
  btn.textContent = "Проверка...";

  try {
    const res = await fetch("/auth/check-email", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    });

    if (res.status === 404) {
      showFieldError("errorEmailText", "errorEmail", "Unable to sign in with this email. Please try another email or create a new account.");
      setInputError("inputEmail", true);
      return;
    }
    if (!res.ok) throw new Error();

    confirmedEmail = email;
    document.getElementById("emailDisplay").textContent = email;
    stepEmail.style.display = "none";
    stepPassword.style.display = "";
    document.getElementById("inputPassword").focus();
  } catch {
    showFieldError("errorEmailText", "errorEmail", "Ошибка сервера. Попробуйте позже.");
    setInputError("inputEmail", true);
  } finally {
    btn.disabled = false;
    btn.textContent = "Далее";
  }
};

// ── Шаг 2: Пароль ────────────────────────────────────────────────────────────

document.getElementById("formPassword").onsubmit = async (e) => {
  e.preventDefault();
  const password = document.getElementById("inputPassword").value;
  const btn      = document.getElementById("btnLogin");

  clearFieldError("errorPassword");
  setInputError("inputPassword", false);

  btn.disabled = true;
  btn.textContent = "Вход...";

  try {
    const res = await fetch("/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: confirmedEmail, password }),
    });
    if (!res.ok) {
      showFieldError("errorPasswordText", "errorPassword", "Неверный пароль.");
      setInputError("inputPassword", true);
      return;
    }
    const data = await res.json();
    localStorage.setItem("token", data.token);
    location.replace("/");
  } catch {
    showFieldError("errorPasswordText", "errorPassword", "Ошибка сервера. Попробуйте позже.");
    setInputError("inputPassword", true);
  } finally {
    btn.disabled = false;
    btn.textContent = "Войти";
  }
};

document.getElementById("btnBack").onclick = () => {
  stepPassword.style.display = "none";
  stepEmail.style.display = "";
  clearFieldError("errorPassword");
  setInputError("inputPassword", false);
  document.getElementById("inputEmail").focus();
};
