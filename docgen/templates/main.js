document.querySelectorAll(".codicon-chevron-right").forEach(el => {
    el.addEventListener("click", (ev) => {
        const par = el.parentElement
        if (par.classList.contains("is-open")) {
            par.classList.remove("is-open")
            par.classList.add("is-closed")
        } else {
            par.classList.add("is-open")
            par.classList.remove("is-closed")
        }
        ev.stopPropagation()
        ev.preventDefault()
    })
})