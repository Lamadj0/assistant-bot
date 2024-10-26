import React from "react";
import './css/footer.css'

export default function Footer(){
    return(
        <div className="main-footer">
            <div className="footer">
                <input type="text" placeholder="Сообщение" />
                <div className="send">
                    <svg xmlns="http://www.w3.org/2000/svg" height="26px" viewBox="0 -960 960 960" width="30px" fill="#f2f2f2"><path d="M440-160v-487L216-423l-56-57 320-320 320 320-56 57-224-224v487h-80Z"/></svg>
                </div>
            </div>
        </div>
    );
}


// function App() {
//   const [question, setQuestion] = useState('');
//   const [answer, setAnswer] = useState('');
//   const [images, setImages] = useState([]);

//   const handleSubmit = async (e) => {
//     e.preventDefault();
//     try {
//       const response = await fetch('http://localhost:8080/ask', {
//         method: 'POST',
//         headers: {
//           'Content-Type': 'application/json',
//         },
//         body: JSON.stringify({ question }),
//       });
//       const data = await response.json();
//       setAnswer(data.answer);
//       setImages(data.images);
//     } catch (error) {
//       console.error('Ошибка при запросе:', error);
//     }
//   };

//   return (
//     <div>
//       <form onSubmit={handleSubmit}>
//         <input
//           type="text"
//           value={question}
//           onChange={(e) => setQuestion(e.target.value)}
//           placeholder="Введите вопрос"
//         />
//         <button type="submit">Спросить</button>
//       </form>
//       {answer && <p>Ответ: {answer}</p>}
//       {images.map((src, index) => (
//         <img key={index} src={src} alt={`Изображение ${index + 1}`} />
//       ))}
//     </div>
//   );
// }

// export default App;