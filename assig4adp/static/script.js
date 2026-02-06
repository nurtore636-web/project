let currentReader = null;
let allBooks = []; // Базадағы кітаптарды сақтау үшін

// 1. Бет жүктелгенде кітаптарды және кестені жүктеу
window.addEventListener("load", async () => {
    await loadBooksToSelect(); // Тізімді толтыру
    await loadLoans();         // Кестені толтыру
});

// 2. Базадан кітаптарды алып, Select-ке салу
async function loadBooksToSelect() {
    try {
        const res = await fetch('/api/books'); // Go-дағы GET /api/books endpoint-і
        if (!res.ok) throw new Error("Кітаптарды алу мүмкін болмады");
        
        allBooks = await res.json();
        const select = document.getElementById('bookSelect');
        
        // Тізімді тазалап, кітаптарды қосу
        select.innerHTML = '<option value="">-- Кітапты таңдаңыз --</option>';
        
        allBooks.forEach(book => {
            let option = document.createElement('option');
            option.value = book.bookId; // ID-ін мән ретінде аламыз
            option.text = book.bookTitle;
            select.appendChild(option);
        });

        // Кітап таңдалғанда автор мен ID-ді автоматты толтыру
        select.addEventListener('change', function() {
            const selectedBook = allBooks.find(b => b.bookId === this.value);
            if (selectedBook) {
                document.getElementById('bookAuthor').value = selectedBook.author;
                document.getElementById('bookId').value = selectedBook.bookId;
            } else {
                document.getElementById('bookAuthor').value = "";
                document.getElementById('bookId').value = "";
            }
        });
    } catch (err) {
        console.error("Қате:", err);
    }
}

function openStep1(){
    document.getElementById("step1").style.display = "block";
    document.getElementById("step2").style.display = "none";
}

function goBack(){
    openStep1();
}

// 3. Оқырманды сақтау
async function saveReader(){
    const name = document.getElementById("readerName").value.trim();
    const email = document.getElementById("readerEmail").value.trim();
    const phone = document.getElementById("readerPhone").value.trim();

    if(!name || !email || !phone){
        alert("Барлық жолақты толтырыңыз!");
        return;
    }

    // currentReader айнымалысына уақытша сақтаймыз
    currentReader = { name, email, phone };

    document.getElementById("step1").style.display = "none";
    document.getElementById("step2").style.display = "block";
}

// 4. Кітап алуды (Loan) сақтау
async function saveLoan(){
    const select = document.getElementById("bookSelect");
    const bookTitle = select.options[select.selectedIndex].text;
    const author = document.getElementById("bookAuthor").value;
    const bookId = document.getElementById("bookId").value;
    const loanDate = document.getElementById("loanDate").value;
    const returnDate = document.getElementById("returnDate").value;

    if(!currentReader){
        alert("Алдымен оқырманды тірке!");
        return;
    }
    if(!bookId || !loanDate || !returnDate){
        alert("Кітапты таңдап, күндерді толтырыңыз!");
        return;
    }

    // Go Backend-тегі LoanRecord құрылымына (struct) сәйкес Payload
    const payload = {
        fullName: currentReader.name,
        email: currentReader.email,
        phone: currentReader.phone,
        bookTitle: bookTitle,
        author: author,
        bookId: bookId,
        loanDate: loanDate,
        returnDate: returnDate
    };

    const resp = await fetch("/api/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
    });

    if(!resp.ok){
        alert("Сақтау сәтсіз аяқталды!");
        return;
    }

    alert("Сәтті сақталды!");
    await loadLoans();
    openStep1();
    
    // Форманы тазалау
    document.querySelectorAll('input').forEach(input => input.value = '');
    select.value = '';
}

// 5. Кестені жаңарту
async function loadLoans(){
    const tbody = document.querySelector("#loanTable tbody");
    tbody.innerHTML = "";

    const res = await fetch("/api/data"); // Барлық жазбаларды алу
    if(!res.ok) return;

    const data = await res.json();
    data.forEach(item => {
        tbody.innerHTML += `
            <tr>
                <td><b>${item.fullName || "-"}</b></td>
                <td>${item.bookTitle} (${item.bookId})</td>
                <td>${item.loanDate} → ${item.returnDate}</td>
            </tr>
        `;
    });
}