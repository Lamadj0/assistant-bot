import { useState } from 'react';
import './css/footer.css';

export default function Footer() {
  const [question, setQuestion] = useState('');
  const [images, setImages] = useState([]);
  const [answers, setAnswers] = useState([]);
  const [showAnswer, setShowAnswer] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();

    const newEntry = { question, answer: '', images: [] };
    setAnswers([...answers, newEntry]);

    try {
      const response = await fetch('http://localhost:8080/ask', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ question }),
      });

      const data = await response.json();
      const updatedEntry = { ...newEntry, answer: data.answer, images: data.images };
      
      setAnswers((prevAnswers) => prevAnswers.map((entry, index) =>
        index === prevAnswers.length - 1 ? updatedEntry : entry
      ));

      setShowAnswer(true);
    } catch (error) {
      console.error('Ошибка при запросе:', error);
    } finally {
      setQuestion('');
    }
  };

  return (
    <>
      <div className="main-footer">
        <div className="scrollable-container">
          {answers.map((item, index) => (
            <div key={index} className="qa-item">
              <p className="question">Вопрос: {item.question}</p>
              {item.answer && <p className="answer">Ответ: {item.answer}</p>}
              {item.images && item.images.map((src, i) => (
                <div className="answer-img" key={i}>
                  <img src={src} alt={`Изображение ${i + 1}`} />
                </div>
              ))}
            </div>
          ))}
        </div>

        <div className="footer">
          <form>
            <input
              type="text"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="Введите сообщение"
            />
            <div type="submit" className="send" onClick={handleSubmit}>
              <svg
                xmlns="http://www.w3.org/2000/svg"
                height="26px"
                viewBox="0 -960 960 960"
                width="30px"
                fill="#f2f2f2"
              >
                <path d="M440-160v-487L216-423l-56-57 320-320 320 320-56 57-224-224v487h-80Z" />
              </svg>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}
