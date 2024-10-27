import { useEffect, useRef, useState } from 'react';
import './css/footer.css';
import './css/header.css'
import logo from './img/Grey.png'
import { Send, X } from 'lucide-react'

export default function Footer() {
  const [question, setQuestion] = useState('');
  const [images, setImages] = useState([]);
  const [answers, setAnswers] = useState([]);
  const [error, setError] = useState('');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [modalImageSrc, setModalImageSrc] = useState('')

  const [dots, setDots] = useState(false);
  const modalDots = () => {
    setDots(!dots)
  }
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

    setError('')
    const newEntry = { question, answer: '', images: [] }
    setAnswers(prevAnswers => [
      ...(Array.isArray(prevAnswers) ? prevAnswers : []),
      newEntry,
    ])

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

  const handleImageClick = (src) => {
    setModalImageSrc(src)
    setIsModalOpen(true)
  };

  const closeModal = () => {
		setModalImageSrc('')
		setIsModalOpen(false)
	}

  // const closeFullscreen = () => {
  //   if (document.exitFullscreen) {
  //     document.exitFullscreen();
  //     setIsFullscreen(false);
  //   }
  // }; чек

  const clearHistory = async () => {
    if (window.confirm('Вы уверены, что хотите очистить историю?')) {
      setAnswers([])
      try {
        await fetch('http://localhost:8080/deleteqa', { method: 'POST' })
      } catch (error) {
        console.error('Ошибка при очистке истории на сервере:', error)
      }
    }
  }

  return (
		<>
    <div className="main-header">
            <div className="header">
                <div className="logo">
                    <img src={logo} alt="" />
                    <div className="name">Бот-Ассистент</div>
                </div>
                <div class="dots" onClick={modalDots}>
                        {dots && (<button className='clear-history' onClick={clearHistory}>
                      Очистить историю
                    </button>)}
                          
                </div>
            </div>
        </div>
			<div className='main-footer'>
				<div className='scrollable-container'>
					{error && <p className='error-message'>{error}</p>}
					{Array.isArray(answers) &&
						answers.map((item, index) => (
							<div key={index} className='qa-item'>
								<p className='question'>{item.question}</p>
								{item.answer && <p className='answer'>{item.answer}</p>}
								{item.images &&
									item.images.map((src, i) => (
										<div className='answer-img' key={i}>
											<img
												src={src}
												alt={`Изображение ${i + 1}`}
												onClick={() => handleImageClick(src)}
											/>
										</div>
									))}
							</div>
						))}
					<div ref={messageRef} />
				</div>

				<div className='footer'>
					<form>
						<input
							type='text'
							value={question}
							onChange={e => setQuestion(e.target.value)}
							onKeyDown={handleKeyDown}
							placeholder='Введите сообщение'
						/>
						<div type='submit' className='send' onClick={handleSubmit}>
							<Send size={22} />
						</div>
					</form>
					
				</div>
			</div>

			{isModalOpen && (
				<div className='modal-overlay' onClick={closeModal}>
					<div className='modal-content' onClick={e => e.stopPropagation()}>
						<img src={modalImageSrc} alt='Просмотр изображения' />
						<button className='close-button' onClick={closeModal}>
							<X />
						</button>
					</div>
				</div>
			)}
		</>
	)
}
