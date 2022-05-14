import 'bulma/css/bulma.css';
import '../bulmaswatch.min.css';
import axios from 'axios';
import { useState } from 'react';

type loginType = {
  Id?: string
  Password?: string
}

export default function Login() {
  const l: loginType = { Id: "", Password: "" }
  const [message, setMessage] = useState("")
  const [dat, setDat] = useState(l)

  const clickSubmit = () => {
    document.querySelector(".notification")?.classList.add("is-hidden")
    axios.post("/auth/start", dat).then(r => window.location.href = "/").catch(r => {
      document.querySelector(".notification")?.classList.remove("is-hidden")
      if(r.response.status == 401) {
        setMessage("ID / PASSWORD do not match.")
      } else if(r.response.status == 400) {
        setMessage("ID / PASSWORD input required.")
      } else {
        setMessage("login failed.")
      }
    })
  }

  return (
    <div className="container">
      <header className="navbar is-dark">
        <div className="navbar-brand">
          <span className="navbar-item has-text-weight-bold is-size-4">modern-utopia.net</span>
          <span className="navbar-item has-text-grey">In the beginning was the Name</span>
        </div>
      </header>
      <div>
        <div className='card px-5'>
          <header className="card-header px-10">
            <p className="card-header-title">
              login
            </p>
          </header>
          <div className="card-content">
            <div className="notification is-danger is-light is-hidden">
              {message}
            </div>
            <div className='content p-10'>
              <div className='field is-horizontal'>
                <div className="field-label is-normal"><label className="label">ID</label></div>
                <div className="field-body">
                  <div className="field">
                    <p className="control">
                      <input type="text" className="input is-primary"
                        value={dat.Id} onChange={e => setDat({ ...dat, Id: e.target.value })} />
                    </p>
                  </div>
                </div>
              </div>
              <div className='field is-horizontal'>
                <div className="field-label is-normal"><label className="label">PASSWORD</label></div>
                <div className="field-body">
                  <div className="field">
                    <p className="control">
                      <input type="password" className="input is-primary is-fullwidth p-2" 
                        value={dat.Password} onChange={e => setDat({ ...dat, Password: e.target.value })} />
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className='card-footer column is-8 is-offset-2'>
            <div className='field'>
              <div className='control'>
                <button type="button" className='button is-primary' onClick={clickSubmit}>submit</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}