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