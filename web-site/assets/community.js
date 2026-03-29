(function () {
  var form = document.getElementById("joinForm");
  if (!form) {
    return;
  }

  var message = document.getElementById("formMessage");

  function setMessage(text, ok) {
    if (!message) {
      return;
    }
    message.textContent = text;
    message.style.color = ok ? "#0d8a58" : "#be3a17";
  }

  function isEmail(value) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
  }

  function readValue(id) {
    var node = document.getElementById(id);
    return node ? node.value.trim() : "";
  }

  form.addEventListener("submit", function (event) {
    event.preventDefault();

    var payload = {
      fullName: readValue("fullName"),
      email: readValue("email"),
      role: readValue("role"),
      focus: readValue("focus"),
      submittedAt: new Date().toISOString()
    };

    if (!payload.fullName || payload.fullName.length < 2) {
      setMessage("Please enter a valid full name.", false);
      return;
    }
    if (!isEmail(payload.email)) {
      setMessage("Please enter a valid email address.", false);
      return;
    }
    if (!payload.role) {
      setMessage("Please select your role.", false);
      return;
    }
    if (!payload.focus || payload.focus.length < 8) {
      setMessage("Please share a short focus area so we can route your request.", false);
      return;
    }

    var storageKey = "aurapanel_community_requests";
    var list = [];
    try {
      var existing = localStorage.getItem(storageKey);
      list = existing ? JSON.parse(existing) : [];
      if (!Array.isArray(list)) {
        list = [];
      }
    } catch (_err) {
      list = [];
    }

    list.push(payload);
    localStorage.setItem(storageKey, JSON.stringify(list));
    form.reset();
    setMessage("Request saved. Thank you. We will contact you through your work email.", true);
  });
})();
