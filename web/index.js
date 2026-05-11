"use strict";

const token = localStorage.getItem("token");

if (token) {
  try {
    const raw = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const payload = JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(raw), c => c.charCodeAt(0))));
    document.getElementById("userEmail").textContent = payload.full_name || payload.user_id || "";
    if (payload.role === "admin") {
      const link = document.createElement("a");
      link.href = "/admin.html";
      link.textContent = "Админка";
      link.style.cssText = "color:var(--danger);font-size:13px;font-weight:700";
      document.getElementById("userEmail").after(link);
    }
  } catch {}
  document.getElementById("btnLogout").style.display = "";
} else {
  document.getElementById("btnLogout").style.display = "none";
  document.getElementById("userEmail").innerHTML =
    '<a href="/login.html" style="color:var(--text);font-weight:700">Войти</a>';
  document.getElementById("btnRegisterNav").classList.add("visible");
}

document.getElementById("btnLogout").onclick = () => {
  localStorage.removeItem("token");
  location.reload();
};

function goToMeet(room) {
  if (!localStorage.getItem("token")) {
    location.href = `/login.html`;
    return;
  }
  location.href = `/meet.html?room=${encodeURIComponent(room)}`;
}

document.getElementById("btnCreate").onclick = () => {
  goToMeet(crypto.randomUUID().slice(0, 8));
};

document.getElementById("btnJoin").onclick = () => {
  const room = document.getElementById("roomInput").value.trim();
  if (!room) { document.getElementById("roomInput").focus(); return; }
  goToMeet(room);
};

document.getElementById("roomInput").addEventListener("keydown", (e) => {
  if (e.key === "Enter") document.getElementById("btnJoin").click();
});
