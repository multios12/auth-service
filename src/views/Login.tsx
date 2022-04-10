import 'bulma/css/bulma.css';
import axios from 'axios';
import { useState } from 'react';

type loginType = {
  Id?: string,
  Password?: string,
  idMessage?: string,
  pwMessage?: string
}

export default function Login() {
  const l: loginType = { Id: "", Password: "" }
  const [dat, setDat] = useState(l)
  const [isErr, setIsErr] = useState(false)

  const clickSubmit = () => {
    axios.post("/auth/start", dat).then(r => window.location.href = "/").catch(r => {
      setIsErr(true)
      if (r.response.status === 404) {
        setDat({ ...dat, idMessage: "The settings are incorrect. Check setting.json and add the user.", pwMessage: undefined })
      } else if (r.response.data.idMessage === undefined && r.response.data.pwMessage === undefined) {
        setDat({ ...dat, idMessage: r.response.data, pwMessage: undefined })
      } else {
        setDat({ ...dat, idMessage: r.response.data.idMessage, pwMessage: r.response.data.pwMessage })
      }
    })
  }

  return (
    <div className='card p-5 is-dark'>
      <header className="card-header">
        <p className="card-header-title">
          login
        </p>
      </header>
      <div className="card-content">
        <div className='content p-5'>
          <div className='field is-horizontal'>
            <div className="field-label is-normal"><label className="label">ID</label></div>
            <div className="field-body">
              <div className="field">
                <p className="control">
                  <input type="text" className="input is-primary" placeholder={dat.idMessage}
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
                  <input type="password" className="input is-primary is-fullwidth p-2" placeholder={dat.pwMessage}
                    value={dat.Password} onChange={e => setDat({ ...dat, Password: e.target.value })} />
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className='card-footer'>
        <div className='field'>
          <div className='control'>
            <button type="button" className='button is-primary' onClick={clickSubmit}>submit</button>
          </div>
        </div>
      </div>
    </div>
  );
}