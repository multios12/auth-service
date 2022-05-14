import { HashRouter as Router, Route, Routes } from 'react-router-dom';
import Login from './views/Login';
import Setting from './views/Setting'

export default function App() {

  return (
    <Router>
      <Routes>
        <Route path='/' element={<Login />} />
        <Route path='/setting' element={<Setting />} />
      </Routes>
    </Router>
  );
}
