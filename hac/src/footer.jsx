import { useEffect, useRef, useState } from 'react';
import './css/footer.css';

export default function Footer() {
  const [question, setQuestion] = useState('');
  const [images, setImages] = useState([]);
  const [answers, setAnswers] = useState([]);
  const [error, setError] = useState('');
  const [isFullscreen, setIsFullscreen] = useState(false);

  const messageRef = useRef(null)
  
  useEffect(() => {
    const fetchHistory = async () => {
      try {
        const res = await fetch('http://localhost:8080/history')
        const data = await res.json()
        setAnswers(data)
      } catch (err) {
        console.log('Ошибка при загрузке истории:', err)
      }
    }

    fetchHistory()
  }, [])

  useEffect(() => {
    messageRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [answers])

  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!question.trim()) {
      setError('Пожалуйста, введите сообщение');
      return;
    }

    setError('');
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
    } catch (error) {
      console.error('Error:', error);
      setError('Не удалось отправить запрос');
    } finally {
      setQuestion('');
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') {
      handleSubmit(e);
    }
  };

  const handleImageClick = async (e) => {
    if (e.target.requestFullscreen) {
      await e.target.requestFullscreen();
      setIsFullscreen(true);
    }
  };

  const closeFullscreen = () => {
    if (document.exitFullscreen) {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  };

  return (
    <>
      <div className="main-footer">
        <div className="scrollable-container">
          {error && <p className="error-message">{error}</p>}
          {answers.map((item, index) => (
            <div key={index} className="qa-item">
              <p className="question">{item.question}</p>
              {item.answer && <p className="answer">{item.answer}</p>}
              {item.images && item.images.map((src, i) => (
                <div className="answer-img" key={i}>
                  <img src={src} alt={`Изображение ${i + 1}`} onClick={handleImageClick} />
                </div>
              ))}
            </div>
          ))}
          <div ref={messageRef} />
        </div>

        <div className="footer">
          <form>
            <input
              type="text"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              onKeyDown={handleKeyDown}
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
