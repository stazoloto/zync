"use strict";

if (localStorage.getItem("token")) location.replace("/");

const screenEmail    = document.getElementById("screenEmail");
const screenCode     = document.getElementById("screenCode");
const screenPassword = document.getElementById("screenPassword");

function showScreen(s) {
    [screenEmail, screenCode, screenPassword].forEach(x => x.style.display = "none");
    s.style.display = "";
}

function showError(id, msg) {
    const el = document.getElementById(id);
    el.textContent = msg;
    el.style.display = "block";
    setTimeout(() => el.style.display = "none", 4000);
}

let pendingEmail = "";
let pendingVerifiedToken = "";

// Шаг 1: email
document.getElementById("formEmail").onsubmit = async (e) => {
    e.preventDefault();
    const email = document.getElementById("inputEmail").value.trim();
    const btn   = document.getElementById("btnSendCode");
    btn.disabled = true;
    btn.textContent = "Отправка...";
    try {
    const res = await fetch("/auth/send-code", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
    });
    if (!res.ok) throw new Error(await res.text());
    pendingEmail = email;
    document.getElementById("codeHint").textContent = `Код отправлен на ${email}`;
    showScreen(screenCode);
    } catch (err) {
    showError("errorEmail", err.message);
    } finally {
    btn.disabled = false;
    btn.textContent = "Продолжить";
    }
};

// Шаг 2: код
document.getElementById("formCode").onsubmit = async (e) => {
    e.preventDefault();
    const code = document.getElementById("inputCode").value.trim();
    const btn  = document.getElementById("btnVerifyCode");
    btn.disabled = true;
    btn.textContent = "Проверка...";
    try {
    const res = await fetch("/auth/verify-code", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: pendingEmail, code }),
    });
    if (!res.ok) throw new Error("Неверный код");
    const data = await res.json();
    pendingVerifiedToken = data.token;
    showScreen(screenPassword);
    } catch (err) {
    showError("errorCode", err.message);
    } finally {
    btn.disabled = false;
    btn.textContent = "Подтвердить";
    }
};

document.getElementById("btnBackToEmail").onclick = (e) => { e.preventDefault(); showScreen(screenEmail); };

// Шаг 3: имя и пароль
document.getElementById("formPassword").onsubmit = async (e) => {
    e.preventDefault();
    const firstName = document.getElementById("inputFirstName").value.trim();
    const lastName  = document.getElementById("inputLastName").value.trim();
    const password  = document.getElementById("inputPassword").value;
    const confirm   = document.getElementById("inputPasswordConfirm").value;

    if (!firstName || !lastName) { showError("errorPassword", "Введите имя и фамилию"); return; }
    if (password !== confirm)    { showError("errorPassword", "Пароли не совпадают"); return; }

    const btn = document.getElementById("btnRegister");
    btn.disabled = true;
    btn.textContent = "Создание аккаунта...";
    try {
    const res = await fetch("/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ verified_token: pendingVerifiedToken, first_name: firstName, last_name: lastName, password }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    localStorage.setItem("token", data.token);
    location.replace("/");
    } catch (err) {
    showError("errorPassword", err.message);
    } finally {
    btn.disabled = false;
    btn.textContent = "Создать аккаунт";
    }
};