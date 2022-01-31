import { Button, Card, CardContent, CardHeader, TextField } from '@mui/material';
import { Box } from '@mui/system';
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

  return <Box m={3}>
    <Card color="primary">
      <CardHeader title="login" />
      <CardContent>
        <TextField name="id" label="id" fullWidth error={isErr} helperText={dat.idMessage}
          value={dat.Id} onChange={e => setDat({ ...dat, Id: e.target.value })} />
        <TextField name="password" label="password" fullWidth error={isErr} helperText={dat.pwMessage}
          value={dat.Password} onChange={e => setDat({ ...dat, Password: e.target.value })} />
        <Button variant="outlined" color="primary" onClick={clickSubmit}>submit</Button>
      </CardContent>
    </Card>
  </Box>;
}