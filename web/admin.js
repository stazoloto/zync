"use strict";

const token = localStorage.getItem("token");
if (!token) { location.replace("/login.html"); throw new Error(); }

try {
  const payload = JSON.parse(atob(token.split(".")[1]));
  if (payload.role !== "admin") { location.replace("/"); throw new Error(); }
} catch { location.replace("/"); }

document.getElementById("btnLogout").onclick = () => {
  localStorage.removeItem("token");
  location.replace("/login.html");
};

function api(method, path, body) {
  return fetch(path, {
    method,
    headers: {
      "Authorization": "Bearer " + token,
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  });
}

// ── Вкладки ───────────────────────────────────────────────────────────────────

document.querySelectorAll(".tab").forEach(tab => {
  tab.onclick = () => {
    document.querySelectorAll(".tab").forEach(t => t.classList.remove("active"));
    tab.classList.add("active");
    const name = tab.dataset.tab;
    document.getElementById("tabUsers").style.display = name === "users" ? "" : "none";
    document.getElementById("tabRooms").style.display = name === "rooms" ? "" : "none";
    if (name === "rooms") loadRooms();
  };
});

// ── Пользователи ─────────────────────────────────────────────────────────────

let allUsers = [];

async function loadUsers() {
  const res = await api("GET", "/admin/users");
  if (!res.ok) {
    document.getElementById("usersBody").innerHTML =
      `<tr><td colspan="5" class="admin-empty">Ошибка загрузки</td></tr>`;
    return;
  }
  allUsers = await res.json();
  applyFilters();
  updateStats();
}

function updateStats() {
  document.getElementById("statTotal").textContent  = allUsers.length;
  document.getElementById("statAdmins").textContent = allUsers.filter(u => u.role === "admin").length;
  document.getElementById("statUsers").textContent  = allUsers.filter(u => u.role === "user").length;
}

function applyFilters() {
  const q    = document.getElementById("searchInput").value.toLowerCase();
  const role = document.getElementById("roleFilter").value;
  const filtered = allUsers.filter(u => {
    const matchSearch = (u.first_name + " " + u.last_name + " " + u.email).toLowerCase().includes(q);
    const matchRole   = !role || u.role === role;
    return matchSearch && matchRole;
  });
  renderUsers(filtered);
}

function renderUsers(users) {
  const body = document.getElementById("usersBody");
  if (!users.length) {
    body.innerHTML = `<tr><td colspan="5" class="admin-empty">Пользователей не найдено</td></tr>`;
    return;
  }
  body.innerHTML = users.map(u => {
    const initials  = (u.first_name[0] + u.last_name[0]).toUpperCase();
    const isAdmin   = u.role === "admin";
    return `
      <tr>
        <td>
          <div class="user-cell">
            <div class="avatar${isAdmin ? " admin-avatar" : ""}">${esc(initials)}</div>
            <span class="user-name">${esc(u.first_name + " " + u.last_name)}</span>
          </div>
        </td>
        <td>${esc(u.email)}</td>
        <td><span class="badge badge-${u.role}">${isAdmin ? "Админ" : "Пользователь"}</span></td>
        <td>${esc(u.created_at)}</td>
        <td>
          <div class="table-actions">
            ${isAdmin
              ? `<button class="btn-sm" onclick="setRole('${u.id}', 'user')">Снять права</button>`
              : `<button class="btn-sm promote" onclick="setRole('${u.id}', 'admin')">Сделать админом</button>`
            }
            <button class="btn-sm danger" onclick="deleteUser('${u.id}')">Удалить</button>
          </div>
        </td>
      </tr>`;
  }).join("");
}

async function deleteUser(id) {
  if (!confirm("Удалить пользователя?")) return;
  const res = await api("DELETE", `/admin/users/${id}`);
  if (res.ok) loadUsers();
  else alert("Ошибка удаления");
}

async function setRole(id, role) {
  const res = await api("PATCH", `/admin/users/${id}/role`, { role });
  if (res.ok) loadUsers();
  else alert("Ошибка смены роли");
}

document.getElementById("searchInput").oninput = applyFilters;
document.getElementById("roleFilter").onchange  = applyFilters;

// ── Комнаты ───────────────────────────────────────────────────────────────────

async function loadRooms() {
  const res = await api("GET", "/admin/rooms");
  if (!res.ok) {
    document.getElementById("roomsBody").innerHTML =
      `<tr><td colspan="4" class="admin-empty">Ошибка загрузки</td></tr>`;
    return;
  }
  const rooms = await res.json();
  document.getElementById("statRooms").textContent = rooms.length;
  renderRooms(rooms);
}

function renderRooms(rooms) {
  const body = document.getElementById("roomsBody");
  if (!rooms.length) {
    body.innerHTML = `<tr><td colspan="4" class="admin-empty">Нет активных комнат</td></tr>`;
    return;
  }
  body.innerHTML = rooms.map(r => {
    const count = r.participants ? r.participants.length : 0;
    const names = count ? r.participants.map(p => `<span class="participant-chip">${esc(p)}</span>`).join("") : "—";
    return `
      <tr>
        <td><span class="room-id">${esc(r.id)}</span></td>
        <td><span class="badge badge-count">${count}</span></td>
        <td><div class="participants-list">${names}</div></td>
        <td>
          <div class="table-actions">
            <a class="btn-sm promote" href="/meet.html?room=${esc(r.id)}" target="_blank">Войти</a>
            <button class="btn-sm danger" onclick="closeRoom('${esc(r.id)}')">Закрыть</button>
          </div>
        </td>
      </tr>`;
  }).join("");
}

async function closeRoom(id) {
  if (!confirm(`Принудительно закрыть комнату ${id}?`)) return;
  const res = await api("DELETE", `/admin/rooms/${id}`);
  if (res.ok) loadRooms();
  else alert("Ошибка закрытия комнаты");
}

document.getElementById("btnRefreshRooms").onclick = loadRooms;

// ── Утилиты ───────────────────────────────────────────────────────────────────

function esc(str) {
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

loadUsers();
