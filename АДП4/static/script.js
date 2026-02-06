let currentReader = null;

function openStep1(){
  document.getElementById("step1").style.display = "block";
  document.getElementById("step2").style.display = "none";
}

function goBack(){
  openStep1();
}

async function saveReader(){
  const name = document.getElementById("readerName").value.trim();
  const email = document.getElementById("readerEmail").value.trim();
  const phone = document.getElementById("readerPhone").value.trim();

  if(!name || !email || !phone){
    alert("Барлық жолақты толтырыңыз!");
    return;
  }

  // Егер сенің Go серверің тек name+phone қабылдаса — email-ды қоспай жібереміз
  const payload = { name, phone, email };

  const resp = await fetch("/add-reader", {
    method: "POST",
    headers: {"Content-Type":"application/json"},
    body: JSON.stringify(payload)
  });

  if(!resp.ok){
    alert("Reader сақталмады: " + await resp.text());
    return;
  }

  currentReader = await resp.json();

  document.getElementById("step1").style.display = "none";
  document.getElementById("step2").style.display = "block";
}

async function saveLoan(){
  const title = document.getElementById("bookTitle").value.trim();
  const author = document.getElementById("bookAuthor").value.trim();
  const bookId = document.getElementById("bookId").value.trim();
  const loanDate = document.getElementById("loanDate").value;
  const returnDate = document.getElementById("returnDate").value;

  if(!currentReader){
    alert("Алдымен оқырманды тірке!");
    return;
  }
  if(!title || !author || !bookId || !loanDate || !returnDate){
    alert("Кітап мәліметтерін толық толтырыңыз!");
    return;
  }

  const payload = {
    reader_name: currentReader.name || currentReader.Name || "Оқырман",
    book_title: title,
    book_id: bookId,
    author,
    loan_date: loanDate,
    return_date: returnDate
  };

  const resp = await fetch("/add-loan", {
    method: "POST",
    headers: {"Content-Type":"application/json"},
    body: JSON.stringify(payload)
  });

  if(!resp.ok){
    alert("Loan сақталмады: " + await resp.text());
    return;
  }

  await loadLoans();
  alert("Сәтті сақталды!");
  openStep1();
}

async function loadLoans(){
  const tbody = document.querySelector("#loanTable tbody");
  tbody.innerHTML = "";

  const res = await fetch("/get-loans");
  if(!res.ok){
    console.log(await res.text());
    return;
  }

  const data = await res.json();
  data.forEach(item=>{
    const reader = item.reader_name || item.ReaderName || "-";
    const book = (item.book_id ? `${item.book_title} (${item.book_id})` : (item.book_title || "-"));
    const term = `${item.loan_date || "-"} → ${item.return_date || "-"}`;

    tbody.innerHTML += `
      <tr>
        <td>${reader}</td>
        <td>${book}</td>
        <td>${term}</td>
      </tr>
    `;
  });
}

window.addEventListener("load", loadLoans);
