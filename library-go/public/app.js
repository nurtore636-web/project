const API = "";
const tokenKey = "lib_token";
const userKey = "lib_user";

function getToken(){ return localStorage.getItem(tokenKey) || ""; }
function setAuth(token, user){
  localStorage.setItem(tokenKey, token);
  localStorage.setItem(userKey, JSON.stringify(user));
}
function clearAuth(){
  localStorage.removeItem(tokenKey);
  localStorage.removeItem(userKey);
}
function me(){ try { return JSON.parse(localStorage.getItem(userKey)||"null"); } catch { return null; } }

async function api(path, method="GET", body=null){
  const headers = { "Content-Type": "application/json" };
  const t = getToken();
  if (t) headers.Authorization = "Bearer " + t;

  const res = await fetch(API + path, { method, headers, body: body?JSON.stringify(body):null });
  const data = await res.json().catch(()=> ({}));
  if (!res.ok) throw new Error(data.error || "Request error");
  return data;
}

function $(id){ return document.getElementById(id); }
function show(el, on=true){ el.style.display = on ? "" : "none"; }


async function initIndex(){
  const meta = await api("/api/meta").catch(()=>({needBootstrapAdmin:false}));
  const needBootstrap = !!meta.needBootstrapAdmin;

  const info = $("bootstrapInfo");
  if (needBootstrap){
    info.textContent = "⚠️ Бірінші админ жоқ. Төменде Admin тіркеп ал (bootstrap).";
    show($("roleRow"), true);
  } else {
    info.textContent = "Reader өздігінен тіркеле алады. Admin жасауды admin ғана жасайды.";
    show($("roleRow"), false);
  }

  $("loginBtn").onclick = async () => {
    $("msg").textContent = "";
    try{
      const email = $("l_email").value.trim();
      const password = $("l_pass").value.trim();
      const r = await api("/api/auth/login","POST",{email,password});
      setAuth(r.token, r.user);
      if (r.user.role === "admin") location.href = "/admin.html";
      else location.href = "/reader.html";
    }catch(e){ $("msg").textContent = e.message; }
  };

  $("regBtn").onclick = async () => {
    $("msg").textContent = "";
    try{
      const fullName = $("r_name").value.trim();
      const email = $("r_email").value.trim();
      const phone = $("r_phone").value.trim();
      const password = $("r_pass").value.trim();
      let role = "reader";
      if (needBootstrap) role = $("r_role").value;

      await api("/api/auth/register","POST",{fullName,email,phone,password,role});
      $("msg").textContent = "✅ Тіркелді. Енді login жаса.";
    }catch(e){ $("msg").textContent = e.message; }
  };
}


function initTopbar(){
  const u = me();
  if ($("who")) $("who").textContent = u ? `${u.fullName} (${u.role})` : "Guest";
  if ($("logout")){
    $("logout").onclick = () => { clearAuth(); location.href = "/"; };
  }
}


async function initAdmin(){
  const u = me();
  if (!u || u.role !== "admin") { location.href="/"; return; }
  initTopbar();

  const tabs = ["books","readers","loans"];
  let current = "books";

  function setTab(t){
    current=t;
    tabs.forEach(x=>{
      $("tab_"+x).classList.toggle("active", x===t);
      show($("pane_"+x), x===t);
    });
    refresh();
  }

  tabs.forEach(t => $("tab_"+t).onclick = ()=>setTab(t));

  async function refresh(){
    $("amsg").textContent = "";
    if (current==="books"){
      const books = await api("/api/books");
      renderBooks(books);
    }
    if (current==="readers"){
      const readers = await api("/api/users?role=reader");
      renderReaders(readers);
    }
    if (current==="loans"){
      const loans = await api("/api/loans");
      renderLoans(loans);
     
      const readers = await api("/api/users?role=reader");
      const books = await api("/api/books");
      fillSelect("borrowReader", readers.map(r=>({id:r.id, label:`${r.fullName} (${r.phone})`})));
      fillSelect("borrowBook", books.map(b=>({id:b.id, label:`${b.bookCode} — ${b.title} (${b.availableQty}/${b.totalQty})`})));
    }
  }

  function fillSelect(id, items){
    const s = $(id);
    s.innerHTML = items.map(x=>`<option value="${x.id}">${x.label}</option>`).join("");
  }

  function renderBooks(books){
    $("booksTable").innerHTML = books.map(b=>`
      <tr>
        <td>${b.bookCode}</td>
        <td>${b.title}</td>
        <td>${b.author}</td>
        <td>${b.price}</td>
        <td>${b.availableQty} / ${b.totalQty}</td>
        <td>
          <button class="secondary" onclick="editBook('${b.id}','${escapeStr(b.title)}','${escapeStr(b.author)}',${b.price},${b.totalQty})">Edit</button>
          <button class="secondary" onclick="delBook('${b.id}')">Delete</button>
        </td>
      </tr>
    `).join("");
  }

  function renderReaders(readers){
    $("readersTable").innerHTML = readers.map(r=>`
      <tr>
        <td>${r.fullName}</td>
        <td>${r.phone}</td>
        <td>${r.email}</td>
        <td><span class="badge">${r.role}</span></td>
        <td>${r.createdAt||""}</td>
      </tr>
    `).join("");
  }

  function renderLoans(loans){
    $("loansTable").innerHTML = loans.map(l=>`
      <tr>
        <td>${l.id}</td>
        <td>${l.readerName||""}</td>
        <td>${l.bookCode||""} — ${l.bookTitle||""}</td>
        <td><span class="badge">${l.status}</span></td>
        <td>${l.loanDate||""}</td>
        <td>${l.dueDate||""}</td>
        <td>${l.returnDate||""}</td>
        <td>${l.fineAmount||0}</td>
        <td>
          ${l.status==="borrowed" ? `
            <button class="secondary" onclick="returnLoan('${l.id}')">Return</button>
            <button class="secondary" onclick="lostLoan('${l.id}')">Lost</button>
          ` : ``}
        </td>
      </tr>
    `).join("");
  }

  function escapeStr(s){ return (s||"").replaceAll("'","&#39;").replaceAll('"',"&quot;"); }

 
  window.delBook = async (id)=>{
    try{ await api("/api/books/"+id,"DELETE"); refresh(); }
    catch(e){ $("amsg").textContent = e.message; }
  };

  window.editBook = (id,title,author,price,totalQty)=>{
    $("eb_id").value=id;
    $("eb_title").value=unescapeHtml(title);
    $("eb_author").value=unescapeHtml(author);
    $("eb_price").value=price;
    $("eb_total").value=totalQty;
    show($("editBox"), true);
  };

  function unescapeHtml(s){
    return s.replaceAll("&#39;","'").replaceAll("&quot;",'"');
  }

  $("closeEdit").onclick = ()=>show($("editBox"), false);

  $("saveEdit").onclick = async ()=>{
    try{
      const id = $("eb_id").value;
      const title = $("eb_title").value.trim();
      const author = $("eb_author").value.trim();
      const price = Number($("eb_price").value||0);
      const totalQty = Number($("eb_total").value||0);
      await api("/api/books/"+id,"PATCH",{title,author,price,totalQty});
      show($("editBox"), false);
      refresh();
    }catch(e){ $("amsg").textContent = e.message; }
  };

  $("addBookBtn").onclick = async ()=>{
    $("amsg").textContent = "";
    try{
      const bookCode = $("b_code").value.trim();
      const title = $("b_title").value.trim();
      const author = $("b_author").value.trim();
      const price = Number($("b_price").value||0);
      const totalQty = Number($("b_total").value||1);
      await api("/api/books","POST",{bookCode,title,author,price,totalQty});
      $("b_code").value=""; $("b_title").value=""; $("b_author").value=""; $("b_price").value=""; $("b_total").value="";
      refresh();
    }catch(e){ $("amsg").textContent = e.message; }
  };

  $("borrowBtn").onclick = async ()=>{
    $("amsg").textContent = "";
    try{
      const readerId = $("borrowReader").value;
      const bookId = $("borrowBook").value;
      const dueDate = $("borrowDue").value.trim();
      await api("/api/loans/borrow","POST",{readerId,bookId,dueDate});
      refresh();
    }catch(e){ $("amsg").textContent = e.message; }
  };

  window.returnLoan = async (loanId)=>{
    try{ await api("/api/loans/return","POST",{loanId}); refresh(); }
    catch(e){ $("amsg").textContent = e.message; }
  };

  window.lostLoan = async (loanId)=>{
    const fine = prompt("Fine amount (KZT). Бос қалдырса — кітап бағасы:");
    try{
      const body = { loanId };
      if (fine && fine.trim() !== "") body.fineAmount = Number(fine);
      await api("/api/loans/lost","POST",body);
      refresh();
    }catch(e){ $("amsg").textContent = e.message; }
  };

  setTab("books");
}


async function initReader(){
  const u = me();
  if (!u || u.role !== "reader") { location.href="/"; return; }
  initTopbar();

  async function refresh(){
    $("rmsg").textContent = "";
    try{
      const books = await api("/api/books");
      $("rBooks").innerHTML = books.map(b=>`
        <tr>
          <td>${b.bookCode}</td>
          <td>${b.title}</td>
          <td>${b.author}</td>
          <td>${b.price}</td>
          <td>${b.availableQty}/${b.totalQty}</td>
        </tr>
      `).join("");

      const my = await api("/api/myloans");
      $("myLoans").innerHTML = my.map(x=>`
        <tr>
          <td>${x.id}</td>
          <td>${x.book?.bookCode||""} — ${x.book?.title||""}</td>
          <td><span class="badge">${x.status}</span></td>
          <td>${x.loanDate||""}</td>
          <td>${x.dueDate||""}</td>
          <td>${x.returnDate||""}</td>
          <td>${x.fineAmount||0}</td>
        </tr>
      `).join("");
    }catch(e){ $("rmsg").textContent = e.message; }
  }

  refresh();
}


window.addEventListener("DOMContentLoaded", ()=>{
  const page = document.body.getAttribute("data-page");
  if (page==="index") initIndex();
  if (page==="admin") initAdmin();
  if (page==="reader") initReader();
});
