import React from 'react';
import ReactDOM from 'react-dom/client';
import Header from './header';
import Footer from './footer'
import Main from './main'


const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <Header />
    <Main />
    <Footer />
  </React.StrictMode>
);

