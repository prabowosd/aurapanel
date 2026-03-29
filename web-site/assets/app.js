(function () {
  var yearNode = document.getElementById("year");
  if (yearNode) {
    yearNode.textContent = String(new Date().getFullYear());
  }

  var progressLine = document.getElementById("progressLine");
  var updateProgress = function () {
    if (!progressLine) {
      return;
    }
    var maxScroll = document.documentElement.scrollHeight - window.innerHeight;
    var ratio = maxScroll > 0 ? window.scrollY / maxScroll : 0;
    progressLine.style.width = Math.min(100, Math.max(0, ratio * 100)).toFixed(2) + "%";
  };

  window.addEventListener("scroll", updateProgress, { passive: true });
  window.addEventListener("resize", updateProgress);
  updateProgress();

  var revealNodes = Array.prototype.slice.call(document.querySelectorAll(".reveal"));
  var revealObserver = new IntersectionObserver(
    function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          entry.target.classList.add("is-visible");
          revealObserver.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.14 }
  );

  revealNodes.forEach(function (node, index) {
    node.style.transitionDelay = Math.min(index * 35, 220) + "ms";
    revealObserver.observe(node);
  });

  var counterNodes = Array.prototype.slice.call(document.querySelectorAll("[data-count]"));
  var animateCounter = function (node) {
    var target = Number(node.getAttribute("data-count") || "0");
    var isFloat = String(target).indexOf(".") >= 0;
    var duration = 1200;
    var start = performance.now();

    var frame = function (now) {
      var progress = Math.min(1, (now - start) / duration);
      var eased = 1 - Math.pow(1 - progress, 3);
      var value = target * eased;
      node.textContent = isFloat ? value.toFixed(1) : String(Math.round(value));
      if (progress < 1) {
        requestAnimationFrame(frame);
      } else {
        node.textContent = isFloat ? target.toFixed(1) : String(target);
      }
    };
    requestAnimationFrame(frame);
  };

  var counterObserver = new IntersectionObserver(
    function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) {
          return;
        }
        animateCounter(entry.target);
        counterObserver.unobserve(entry.target);
      });
    },
    { threshold: 0.45 }
  );

  counterNodes.forEach(function (node) {
    counterObserver.observe(node);
  });
})();
