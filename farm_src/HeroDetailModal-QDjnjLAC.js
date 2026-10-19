import{g as d}from"./index-tLx1ZRcz.js";import{h}from"./hero-portraits-DHdUCmRS.js";import{U as n,h as u}from"./icons-DKqKqHsB.js";import{C as $}from"./class-presentation-DT0MRBXk.js";const g=["weapon","armor","tool","boots","ring","necklace"],c={weapon:"Arma",armor:"Armadura",tool:"Ferramenta",boots:"Botas",ring:"Anel",necklace:"Colar"},b={weapon:"⚔",armor:"◆",tool:"⚒",boots:"♟",ring:"◉",necklace:"◇"},s=t=>Math.round(t||0).toLocaleString(d());function y(t){const a=[];return t.stats.hp&&a.push(`HP +${s(t.stats.hp)}`),t.stats.damage&&a.push(`ATK +${s(t.stats.damage)}`),t.stats.speed&&a.push(`SPD +${s(t.stats.speed)}`),t.stats.collection&&a.push(`HARV +${s(t.stats.collection)}`),t.stats.armor&&a.push(`ARM +${s(t.stats.armor)}`),t.stats.criticalChance&&a.push(`CRIT +${s(t.stats.criticalChance)}%`),t.stats.damageReduction&&a.push(`RED +${s(t.stats.damageReduction)}%`),t.stats.dodgeChance&&a.push(`ESQ +${s(t.stats.dodgeChance)}%`),a.join(" · ")}class f{dialog;title;body;ensure(){if(this.dialog)return;const a=document.createElement("dialog");a.id="hero-detail-modal",a.setAttribute("aria-labelledby","hero-detail-title"),a.innerHTML=`
      <header><div><small>HERÓI</small><h1 id="hero-detail-title">Herói</h1></div><button type="button" class="hero-detail-close" aria-label="Fechar">×</button></header>
      <section class="hero-detail-body"></section>`,document.querySelector("#game-shell").append(a),a.querySelector(".hero-detail-close").addEventListener("click",()=>a.close()),this.title=a.querySelector("#hero-detail-title"),this.body=a.querySelector(".hero-detail-body"),this.dialog=a}open(a){this.ensure(),this.title.textContent=a.name;const i=h(a.code,a.rarity),r=$[a.code],o=a.stats||{hp:0,damage:0,speed:0,collection:0},p=g.map(l=>{const e=(a.items||[]).find(m=>m.slot===l);return e?`<article class="equipment-slot equipment-${l} filled" style="--rarity:${e.rarityColor||"#725039"}"><span>${c[l]}</span><img src="${e.sprite}" alt="${e.name}"><strong>${e.name}</strong><small><span class="rarity-text">${e.rarityName}</span><br>${y(e)||"Sem atributos"}</small></article>`:`<article class="equipment-slot equipment-${l}"><span>${c[l]}</span><i class="empty-equipment-icon">${b[l]}</i><em>Vazio</em></article>`}).join("");this.body.innerHTML=`
      <div class="hero-detail-hero">
        <span class="owned-portrait hero-detail-portrait">${i.url?`<img class="${i.custom?"custom-hero-portrait":""}" src="${i.url}" alt="">`:"♟"}</span>
        <div class="hero-detail-meta">
          <small class="ranking-rarity" data-rarity="${a.rarity}">${a.rarityName}</small>
          <p>Nível ${a.level}</p>
          ${u(a.stars)}
          <b class="hero-detail-power">⚡ POWER ${(a.power||0).toLocaleString(d())}</b>
          ${r?`<p class="hero-detail-skill" title="${r.full}">${r.short}</p>`:""}
        </div>
      </div>
      <div class="hero-detail-stats">
        <span><img src="/icones-hud/coracao.png" alt=""><small>HP</small><b>${s(o.hp)}</b></span>
        <span><img src="${n.attack}" alt=""><small>ATK</small><b>${s(o.damage)}</b></span>
        <span><img src="${n.speed}" alt=""><small>SPD</small><b>${s(o.speed)}</b></span>
        <span><img src="${n.collection}" alt=""><small>HARV</small><b>${s(o.collection)}</b></span>
      </div>
      <div class="hero-detail-items">${p}</div>`,this.dialog.open||this.dialog.showModal()}}const S=new f,q=t=>S.open(t);export{q as o};
