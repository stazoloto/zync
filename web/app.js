"use strict";

if (!localStorage.getItem("token")) {
  location.replace("/login.html");
  throw new Error("unauthenticated");
}

// ── DOM refs ─────────────────────────────────────────────────────────────────

const mainVideo      = document.getElementById("mainVideo");
const inviteOverlay  = document.getElementById("inviteOverlay");
const inviteRoom     = document.getElementById("inviteRoom");
const filmstrip      = document.getElementById("filmstrip");
const participantCount = document.getElementById("participantCount");
const participantsBlock  = document.getElementById("participantsBlock");
const chatPanel      = document.getElementById("chatPanel");
const chatLog        = document.getElementById("chatLog");
const chatEmpty      = document.getElementById("chatEmpty");
const chatForm       = document.getElementById("chatForm");
const chatInput      = document.getElementById("chatInput");
const chatBadge      = document.getElementById("chatBadge");
const chatToast      = document.getElementById("chatToast");
const participantsPanel  = document.getElementById("participantsPanel");
const participantsList   = document.getElementById("participantsList");
const participantsPanelCount = document.getElementById("participantsPanelCount");

// ── State ─────────────────────────────────────────────────────────────────────

let room         = "";
let clientId     = "";
let myName       = "";
let ws           = null;
let pc           = null;
let localStream  = null;
let screenStream = null;
let makingOffer  = false;
let audioEnabled = true;
let videoEnabled = true;
let sharingScreen = false;
let chatOpen     = false;
let participantsOpen = false;
let unreadCount  = 0;
let spotlightId  = "self";
let currentLayout = "view"; // view | strip | compact

const polite            = true;
const pendingCandidates = [];
const remoteTiles       = new Map(); // streamId → { tile, stream }
const remoteAudios      = new Map();

// ── Layout switching ──────────────────────────────────────────────────────────

function setLayout(layout) {
  currentLayout = layout;
  participantsBlock.className = `participants-block layout-${layout}`;

  document.querySelectorAll(".layout-btn").forEach(btn => {
    btn.classList.toggle("active", btn.dataset.layout === layout);
  });

  // In strip (PiP) mode show only first tile; view and compact show all
  filmstrip.querySelectorAll(".film-tile").forEach((tile, i) => {
    tile.style.display = (layout === "strip" && i > 0) ? "none" : "";
  });
}

document.querySelectorAll(".layout-btn").forEach(btn => {
  btn.onclick = () => setLayout(btn.dataset.layout);
});

// ── Spotlight ─────────────────────────────────────────────────────────────────

function setSpotlight(id) {
  spotlightId = id;

  document.querySelectorAll(".film-tile").forEach(t => t.classList.remove("selected"));
  const activeTile = document.querySelector(`.film-tile[data-id="${id}"]`);
  if (activeTile) activeTile.classList.add("selected");

  if (id === "self") {
    mainVideo.srcObject = localStream;
  } else {
    const entry = remoteTiles.get(id);
    if (entry) {
      mainVideo.srcObject = entry.stream;
    }
  }
}

// ── Filmstrip ─────────────────────────────────────────────────────────────────

function updateParticipantCount() {
  const total = 1 + remoteTiles.size;
  participantCount.textContent = total;
  participantsPanelCount.textContent = total;
  if (participantsOpen) renderParticipantsPanel();
}

function createSelfTile() {
  const tile = document.createElement("article");
  tile.className = "film-tile selected";
  tile.dataset.id = "self";

  const avatar = document.createElement("div");
  avatar.className = "film-avatar";
  avatar.textContent = initials(myName);

  const video = document.createElement("video");
  video.autoplay    = true;
  video.muted       = true;
  video.playsInline = true;

  const me = document.createElement("span");
  me.className = "film-me-chip";
  me.textContent = "Я";

  const label = document.createElement("span");
  label.className = "film-label";
  label.textContent = myName;

  tile.append(avatar, video, me, label);
  tile.onclick = () => setSpotlight("self");
  filmstrip.appendChild(tile);
  return video;
}

function addRemoteTile(streamId, stream) {
  if (remoteTiles.has(streamId)) return;

  const tile = document.createElement("article");
  tile.className = "film-tile";
  tile.dataset.id = streamId;

  // In compact mode, hide all non-first tiles
  if (currentLayout === "compact" && filmstrip.children.length > 0) {
    tile.style.display = "none";
  }

  const avatar = document.createElement("div");
  avatar.className = "film-avatar";
  avatar.textContent = "U";

  const video = document.createElement("video");
  video.autoplay    = true;
  video.muted       = true;
  video.playsInline = true;
  video.srcObject   = stream;

  const label = document.createElement("span");
  label.className = "film-label";
  label.textContent = "Участник";

  tile.append(avatar, video, label);
  tile.onclick = () => setSpotlight(streamId);
  filmstrip.appendChild(tile);

  remoteTiles.set(streamId, { tile, stream });
  updateParticipantCount();
  updateFilmNav();
  inviteOverlay.style.display = "none";

  if (spotlightId === "self") setSpotlight(streamId);
}

function removeRemoteTile(streamId) {
  const entry = remoteTiles.get(streamId);
  if (!entry) return;
  entry.tile.remove();
  remoteTiles.delete(streamId);
  updateParticipantCount();
  updateFilmNav();

  if (remoteTiles.size === 0) {
    inviteOverlay.style.display = "";
    setSpotlight("self");
  } else if (spotlightId === streamId) {
    setSpotlight(remoteTiles.keys().next().value);
  }
}

function initials(name) {
  const parts = name.trim().split(/\s+/);
  return parts.length >= 2
    ? (parts[0][0] + parts[1][0]).toUpperCase()
    : name.slice(0, 2).toUpperCase();
}

// ── Participants list panel ───────────────────────────────────────────────────

function renderParticipantsPanel() {
  participantsList.innerHTML = "";

  const selfRow = document.createElement("div");
  selfRow.className = "participant-row";
  selfRow.innerHTML = `
    <div class="participant-avatar self">${initials(myName)}</div>
    <div class="participant-info">
      <div class="participant-name">${esc(myName)}</div>
      <div class="participant-you">Вы</div>
    </div>
    <img class="participant-mic" src="/static/assets/icons/${audioEnabled ? "Audio" : "AudioOff"}.svg" alt="" />
  `;
  participantsList.appendChild(selfRow);

  remoteTiles.forEach(() => {
    const row = document.createElement("div");
    row.className = "participant-row";
    row.innerHTML = `
      <div class="participant-avatar">U</div>
      <div class="participant-info">
        <div class="participant-name">Участник</div>
      </div>
      <img class="participant-mic" src="/static/assets/icons/Audio.svg" alt="" />
    `;
    participantsList.appendChild(row);
  });
}

function toggleParticipants() {
  participantsOpen = !participantsOpen;
  participantsPanel.classList.toggle("open", participantsOpen);
  document.getElementById("btnParticipants").classList.toggle("share-on", participantsOpen);
  if (participantsOpen) renderParticipantsPanel();
}

document.getElementById("btnCloseParticipants").onclick = toggleParticipants;

// ── Filmstrip navigation ──────────────────────────────────────────────────────

const btnFilmPrev = document.getElementById("btnFilmPrev");
const btnFilmNext = document.getElementById("btnFilmNext");

function updateFilmNav() {
  const atStart = filmstrip.scrollLeft <= 0;
  const atEnd   = filmstrip.scrollLeft + filmstrip.clientWidth >= filmstrip.scrollWidth - 1;
  btnFilmPrev.style.visibility = atStart ? "hidden" : "";
  btnFilmNext.style.visibility = atEnd   ? "hidden" : "";
}

btnFilmPrev.onclick = () => {
  filmstrip.scrollBy({ left: -160, behavior: "smooth" });
};

btnFilmNext.onclick = () => {
  filmstrip.scrollBy({ left: 160, behavior: "smooth" });
};

filmstrip.addEventListener("scroll", updateFilmNav);
updateFilmNav();

// ── WebSocket helpers ─────────────────────────────────────────────────────────

function wsURL() {
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  const token  = localStorage.getItem("token") || "";
  const params = new URLSearchParams({ client_id: clientId, room_id: room, token });
  return `${scheme}://${location.host}/ws?${params.toString()}`;
}

function send(type, payload = {}, to = "sfu") {
  if (!ws || ws.readyState !== WebSocket.OPEN) { console.warn("[send] ws not open", ws?.readyState); return; }
  ws.send(JSON.stringify({ type, room, from: clientId, to, payload }));
}

function flushCandidates() {
  if (!pc?.remoteDescription) return;
  while (pendingCandidates.length) {
    pc.addIceCandidate(pendingCandidates.shift()).catch(console.warn);
  }
}

function safePlay(el) { el.play().catch(() => {}); }

// ── Chat ──────────────────────────────────────────────────────────────────────

let toastTimer = null;

function showToast(name, text) {
  chatToast.innerHTML = `<div class="chat-toast-name">${esc(name)}</div><div>${esc(text)}</div>`;
  chatToast.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => chatToast.classList.remove("show"), 3500);
}

let lastChatSender = null;

function appendChat(chat) {
  chatEmpty.style.display = "none";

  const displayName = chat.from_name || chat.from || "unknown";
  const isOwn = displayName === myName;
  const showHeader = displayName !== lastChatSender;
  lastChatSender = displayName;
  const date  = chat.created_at ? new Date(chat.created_at) : new Date();
  const timeStr = date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });

  const row = document.createElement("div");
  row.className = `message ${isOwn ? "own" : "other"}`;

  // Шапка: только для первого сообщения в серии
  if (showHeader) {
    const header = document.createElement("div");
    header.className = "message-header";

    const avatar = document.createElement("div");
    avatar.className = "message-avatar";
    avatar.textContent = initials(displayName);

    const author = document.createElement("div");
    author.className = "message-author";
    author.textContent = displayName;

    header.appendChild(avatar);
    header.appendChild(author);
    row.appendChild(header);
  }

  // Пузырь + время
  const body = document.createElement("div");
  body.className = "message-body";

  if (chat.text) {
    const bubble = document.createElement("div");
    bubble.className = "message-bubble";
    bubble.textContent = chat.text;
    body.appendChild(bubble);
  }

  const time = document.createElement("div");
  time.className = "message-time";
  time.textContent = timeStr;
  body.appendChild(time);

  row.appendChild(body);
  chatLog.appendChild(row);
  chatLog.scrollTop = chatLog.scrollHeight;

  if (!chatOpen) {
    unreadCount++;
    chatBadge.classList.add("visible");
    showToast(displayName, chat.text || "");
  }
}


function toggleChat() {
  chatOpen = !chatOpen;
  chatPanel.classList.toggle("open", chatOpen);
  document.querySelector(".meet").classList.toggle("chat-open", chatOpen);
  document.getElementById("btnChat").classList.toggle("share-on", chatOpen);
  if (chatOpen) {
    unreadCount = 0;
    chatBadge.classList.remove("visible");
    chatInput.focus();
  }
}

document.getElementById("btnChat").onclick      = toggleChat;
document.getElementById("btnCloseChat").onclick  = toggleChat;

chatForm.onsubmit = (e) => {
  e.preventDefault();
  const text = chatInput.value.trim();
  if (!text) return;
  send("chat", { text, from_name: myName }, "");
  chatInput.value = "";
  chatInput.focus();
};

// ── Emoji picker ──────────────────────────────────────────────────────────────

const EMOJIS = {
  "😀": "смайлики", "😁": "смайлики", "😂": "смайлики", "🤣": "смайлики", "😃": "смайлики",
  "😄": "смайлики", "😅": "смайлики", "😆": "смайлики", "😉": "смайлики", "😊": "смайлики",
  "😋": "смайлики", "😎": "смайлики", "😍": "смайлики", "🥰": "смайлики", "😘": "смайлики",
  "😗": "смайлики", "😙": "смайлики", "😚": "смайлики", "🙂": "смайлики", "🤗": "смайлики",
  "🤩": "смайлики", "🤔": "смайлики", "🤨": "смайлики", "😐": "смайлики", "😑": "смайлики",
  "😶": "смайлики", "🙄": "смайлики", "😏": "смайлики", "😣": "смайлики", "😥": "смайлики",
  "😮": "смайлики", "🤐": "смайлики", "😯": "смайлики", "😪": "смайлики", "😫": "смайлики",
  "😴": "смайлики", "😌": "смайлики", "😛": "смайлики", "😜": "смайлики", "😝": "смайлики",
  "🤤": "смайлики", "😒": "смайлики", "😓": "смайлики", "😔": "смайлики", "😕": "смайлики",
  "🙃": "смайлики", "🤑": "смайлики", "😲": "смайлики", "🙁": "смайлики", "😖": "смайлики",
  "😞": "смайлики", "😟": "смайлики", "😤": "смайлики", "😢": "смайлики", "😭": "смайлики",
  "😦": "смайлики", "😧": "смайлики", "😨": "смайлики", "😩": "смайлики", "🤯": "смайлики",
  "😬": "смайлики", "😰": "смайлики", "😱": "смайлики", "🥵": "смайлики", "🥶": "смайлики",
  "😳": "смайлики", "🤪": "смайлики", "😵": "смайлики", "🥴": "смайлики", "😷": "смайлики",
  "🤒": "смайлики", "🤕": "смайлики", "🤢": "смайлики", "🤮": "смайлики", "🤧": "смайлики",
  "😇": "смайлики", "🥳": "смайлики", "🥺": "смайлики", "🤠": "смайлики", "🤡": "смайлики",
  "🤥": "смайлики", "🤫": "смайлики", "🤭": "смайлики", "🧐": "смайлики", "🤓": "смайлики",
  "😈": "смайлики", "👿": "смайлики", "👹": "смайлики", "👺": "смайлики", "💀": "смайлики",
  "👻": "смайлики", "👽": "смайлики", "🤖": "смайлики", "💩": "смайлики", "😺": "смайлики",

  "👍": "жесты", "👎": "жесты", "👌": "жесты", "✌️": "жесты", "🤞": "жесты",
  "🤟": "жесты", "🤘": "жесты", "🤙": "жесты", "👈": "жесты", "👉": "жесты",
  "👆": "жесты", "👇": "жесты", "☝️": "жесты", "✋": "жесты", "🤚": "жесты",
  "🖐": "жесты", "🖖": "жесты", "👋": "жесты", "🤙": "жесты", "💪": "жесты",
  "🦵": "жесты", "🦶": "жесты", "👏": "жесты", "🙌": "жесты", "🤲": "жесты",
  "🙏": "жесты", "✍️": "жесты", "💅": "жесты", "🤳": "жесты", "💏": "жесты",

  "❤️": "сердца", "🧡": "сердца", "💛": "сердца", "💚": "сердца", "💙": "сердца",
  "💜": "сердца", "🖤": "сердца", "🤍": "сердца", "🤎": "сердца", "💔": "сердца",
  "❣️": "сердца", "💕": "сердца", "💞": "сердца", "💓": "сердца", "💗": "сердца",
  "💖": "сердца", "💘": "сердца", "💝": "сердца", "💟": "сердца", "☮️": "сердца",

  "🎉": "праздник", "🎊": "праздник", "🎈": "праздник", "🎁": "праздник", "🎂": "праздник",
  "🍰": "праздник", "🥂": "праздник", "🍾": "праздник", "🥳": "праздник", "🎆": "праздник",
  "🎇": "праздник", "✨": "праздник", "🎀": "праздник", "🎗": "праздник", "🏆": "праздник",
  "🥇": "праздник", "🥈": "праздник", "🥉": "праздник", "🎖": "праздник", "🎯": "праздник",

  "🐶": "животные", "🐱": "животные", "🐭": "животные", "🐹": "животные", "🐰": "животные",
  "🦊": "животные", "🐻": "животные", "🐼": "животные", "🐨": "животные", "🐯": "животные",
  "🦁": "животные", "🐮": "животные", "🐷": "животные", "🐸": "животные", "🐵": "животные",
  "🐔": "животные", "🐧": "животные", "🐦": "животные", "🦆": "животные", "🦅": "животные",
  "🦉": "животные", "🦇": "животные", "🐝": "животные", "🐛": "животные", "🦋": "животные",

  "🍎": "еда", "🍊": "еда", "🍋": "еда", "🍇": "еда", "🍓": "еда",
  "🍒": "еда", "🍑": "еда", "🥭": "еда", "🍍": "еда", "🥥": "еда",
  "🍕": "еда", "🍔": "еда", "🍟": "еда", "🌭": "еда", "🍿": "еда",
  "🍦": "еда", "🍩": "еда", "🍪": "еда", "☕": "еда", "🍵": "еда",
  "🧋": "еда", "🥤": "еда", "🍺": "еда", "🍻": "еда", "🥃": "еда",
};

const CATS = [
  { icon: "😀", label: "смайлики" },
  { icon: "👍", label: "жесты" },
  { icon: "❤️", label: "сердца" },
  { icon: "🎉", label: "праздник" },
  { icon: "🐶", label: "животные" },
  { icon: "🍎", label: "еда" },
];

const btnEmoji    = document.getElementById("btnEmoji");
const emojiPicker = document.getElementById("emojiPicker");
const emojiGrid   = document.getElementById("emojiGrid");
const emojiSearch = document.getElementById("emojiSearch");
const emojiCats   = document.getElementById("emojiCats");
let   emojiOpen   = false;
let   activeCat   = "смайлики";

function renderEmojiGrid(filter = "") {
  emojiGrid.innerHTML = "";
  const q = filter.toLowerCase();
  for (const [emoji, cat] of Object.entries(EMOJIS)) {
    if (filter ? emoji.includes(q) || cat.includes(q) : cat === activeCat) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "emoji-btn";
      btn.textContent = emoji;
      btn.onclick = () => insertEmoji(emoji);
      emojiGrid.appendChild(btn);
    }
  }
}

function renderCats() {
  emojiCats.innerHTML = "";
  CATS.forEach(({ icon, label }) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "emoji-cat-btn" + (label === activeCat ? " active" : "");
    btn.textContent = icon;
    btn.title = label;
    btn.onclick = () => {
      activeCat = label;
      emojiSearch.value = "";
      renderCats();
      renderEmojiGrid();
    };
    emojiCats.appendChild(btn);
  });
}

function insertEmoji(emoji) {
  const pos = chatInput.selectionStart ?? chatInput.value.length;
  chatInput.value = chatInput.value.slice(0, pos) + emoji + chatInput.value.slice(pos);
  chatInput.focus();
  chatInput.selectionStart = chatInput.selectionEnd = pos + emoji.length;
}

btnEmoji.onclick = (e) => {
  e.stopPropagation();
  emojiOpen = !emojiOpen;
  if (emojiOpen) {
    renderCats();
    renderEmojiGrid();
  }
  emojiPicker.classList.toggle("open", emojiOpen);
};

emojiSearch.oninput = () => renderEmojiGrid(emojiSearch.value);

emojiPicker.addEventListener("click", (e) => e.stopPropagation());

document.addEventListener("click", (e) => {
  if (emojiOpen && e.target !== btnEmoji && !btnEmoji.contains(e.target)) {
    emojiOpen = false;
    emojiPicker.classList.remove("open");
  }
});

// ── Invite ────────────────────────────────────────────────────────────────────

document.getElementById("btnInvite").onclick   = copyLink;
document.getElementById("btnCopyLink").onclick  = copyLink;

async function copyLink() {
  const url  = `${location.origin}/meet.html?room=${encodeURIComponent(room)}`;
  const text = document.getElementById("copyLinkText");
  try {
    await navigator.clipboard.writeText(url);
    text.textContent = "Скопировано!";
    setTimeout(() => text.textContent = "Копировать ссылку", 2000);
  } catch {
    prompt("Скопируйте ссылку:", url);
  }
}

// ── Controls ──────────────────────────────────────────────────────────────────

document.getElementById("btnMute").onclick = () => {
  audioEnabled = !audioEnabled;
  localStream?.getAudioTracks().forEach(t => t.enabled = audioEnabled);
  document.getElementById("btnMute").classList.toggle("muted", !audioEnabled);
  document.getElementById("micIcon").src = audioEnabled
    ? "/static/assets/icons/Audio.svg"
    : "/static/assets/icons/AudioOff.svg";
  if (participantsOpen) renderParticipantsPanel();
};

document.getElementById("btnCam").onclick = () => {
  videoEnabled = !videoEnabled;
  localStream?.getVideoTracks().forEach(t => t.enabled = videoEnabled);
  document.getElementById("btnCam").classList.toggle("muted", !videoEnabled);
  document.getElementById("camIcon").src = videoEnabled
    ? "/static/assets/icons/Camera.svg"
    : "/static/assets/icons/CameraOff.svg";
};

document.getElementById("btnShare").onclick = async () => {
  const btn = document.getElementById("btnShare");
  if (sharingScreen) {
    screenStream?.getTracks().forEach(t => t.stop());
    screenStream = null;
    sharingScreen = false;
    btn.classList.remove("share-on");
    const videoTrack = localStream?.getVideoTracks()[0];
    if (videoTrack) {
      pc?.getSenders().find(s => s.track?.kind === "video")?.replaceTrack(videoTrack);
    }
    return;
  }
  try {
    screenStream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: false });
    const track  = screenStream.getVideoTracks()[0];
    await pc?.getSenders().find(s => s.track?.kind === "video")?.replaceTrack(track);
    sharingScreen = true;
    btn.classList.add("share-on");
    track.onended = () => document.getElementById("btnShare").click();
  } catch {}
};

document.getElementById("btnLeave").onclick = () => {
  localStream?.getTracks().forEach(t => t.stop());
  screenStream?.getTracks().forEach(t => t.stop());
  pc?.close();
  ws?.close();
  location.replace("/");
};

// ── WebRTC ────────────────────────────────────────────────────────────────────

async function start() {
  ws = new WebSocket(wsURL());

  ws.onopen = async () => {
    send("join", {});

    pc = new RTCPeerConnection({ iceServers: [{ urls: "stun:stun.l.google.com:19302" }] });

    pc.onicecandidate = (e) => { if (e.candidate) send("candidate", e.candidate); };

    pc.ontrack = (e) => {
      const streamId = e.streams[0]?.id;
      if (e.track.kind === "audio") {
        if (remoteAudios.has(streamId)) return;
        const audio = document.createElement("audio");
        audio.autoplay  = true;
        audio.srcObject = new MediaStream([e.track]);
        audio.style.display = "none";
        document.body.appendChild(audio);
        remoteAudios.set(streamId, audio);
        safePlay(audio);
        e.track.onended = () => { audio.remove(); remoteAudios.delete(streamId); };
      }
      if (e.track.kind === "video") {
        const stream = e.streams[0] || new MediaStream([e.track]);
        addRemoteTile(streamId, stream);
        e.track.onended = () => removeRemoteTile(streamId);
      }
    };

    localStream = await navigator.mediaDevices.getUserMedia({
      video: { width: 960, height: 540, frameRate: 24 },
      audio: true,
    });

    const selfTileVideo = createSelfTile();
    selfTileVideo.srcObject = localStream;
    safePlay(selfTileVideo);

    mainVideo.srcObject = localStream;

    for (const track of localStream.getTracks()) pc.addTrack(track, localStream);

    const videoSender = pc.getSenders().find(s => s.track?.kind === "video");
    if (videoSender) {
      try {
        const params = videoSender.getParameters();
        if (!params.encodings?.length) params.encodings = [{}];
        params.encodings[0].maxBitrate = 900_000;
        await videoSender.setParameters(params);
      } catch {}
    }

    try {
      makingOffer = true;
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      send("offer", pc.localDescription);
    } finally { makingOffer = false; }
  };

  ws.onmessage = async (e) => {
    const msg = JSON.parse(e.data);

    if (msg.type === "chat") { appendChat(msg.payload || {}); return; }

    if (msg.type === "peer_left") {
      for (const id of (msg.payload?.stream_ids || [])) {
        removeRemoteTile(id);
        remoteAudios.get(id)?.remove();
        remoteAudios.delete(id);
      }
      return;
    }

    if (msg.type === "offer") {
      const collision = makingOffer || pc.signalingState !== "stable";
      if (collision) { if (!polite) return; await pc.setLocalDescription({ type: "rollback" }); }
      await pc.setRemoteDescription(msg.payload);
      flushCandidates();
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      send("answer", pc.localDescription);
      return;
    }

    if (msg.type === "answer") {
      if (pc.signalingState === "have-local-offer") {
        await pc.setRemoteDescription(msg.payload);
        flushCandidates();
      }
      return;
    }

    if (msg.type === "candidate") {
      if (!pc.remoteDescription) { pendingCandidates.push(msg.payload); return; }
      await pc.addIceCandidate(msg.payload).catch(console.warn);
    }
  };

  ws.onclose = () => {};
  ws.onerror = () => {};
}

// ── Utilities ─────────────────────────────────────────────────────────────────

function esc(str) {
  return String(str).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// ── Enter room ────────────────────────────────────────────────────────────────

function enterRoom(name, roomID, userId) {
  myName   = name;
  clientId = userId || crypto.randomUUID();
  room     = roomID;

  inviteRoom.textContent = room;

  start().catch(err => { console.error(err); });
}

try {
  const raw = localStorage.getItem("token").split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
  const payload = JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(raw), c => c.charCodeAt(0))));
  const name    = payload.full_name || "";
  const userId  = payload.user_id  || payload.sub || "";
  const urlRoom = new URLSearchParams(location.search).get("room") || "room1";
  if (name) enterRoom(name, urlRoom, userId);
} catch {
  location.replace("/login.html");
}
