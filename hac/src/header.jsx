import React from "react";
import './css/header.css'
import logo from './img/Grey.png'


export default function Header(){
    return(
        <div className="main-header">
            <div className="header">
                <div className="logo">
                    <img src={logo} alt="" />
                    <div className="name">Бот-Ассистент</div>
                </div>
                <div class="dots">
                    <span></span>
                </div>
            </div>
        </div>
    );
}