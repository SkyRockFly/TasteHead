function setActiveTab(tabID) {
    document.querySelectorAll(".tab-panel").forEach(p => p.classList.remove("active"))

    document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"))

    const panel = document.getElementById(tabID)
    const btn = document.querySelector(`.tab-btn[data-tab="${tabID}"]`)

    if (panel) panel.classList.add("active")
    if (btn) btn.classList.add("active")

    localStorage.setItem("activeTab",tabID)

}

function initTabs(){
    document.querySelectorAll(".tab-btn").forEach(btn => {
        btn.addEventListener("click", () => setActiveTab(btn.dataset.tab))
    });

    const saved = localStorage.getItem("activeTab")
    if (saved) setActiveTab(saved)
}

window.addEventListener("DOMContentLoaded",initTabs)