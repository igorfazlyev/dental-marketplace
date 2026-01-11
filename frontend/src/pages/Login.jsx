import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Form, Button, Card, Alert } from 'react-bootstrap';
import { useAuth } from '../context/AuthContext';

const Login = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const user = await login(email, password);
      
      if (user.role === 'patient') {
        navigate('/patient');
      } else if (user.role === 'clinic') {
        navigate('/clinic');
      } else if (user.role === 'government') {
        navigate('/government');
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Ошибка входа');
    } finally {
      setLoading(false);
    }
  };

  const quickLogin = (role) => {
    const credentials = {
      patient: { email: 'patient@example.com', password: 'password123' },
      clinic: { email: 'clinic@example.com', password: 'password123' },
      government: { email: 'gov@example.com', password: 'password123' },
    };

    setEmail(credentials[role].email);
    setPassword(credentials[role].password);
  };

  return (
    <div className="d-flex justify-content-center align-items-center" style={{ minHeight: '80vh' }}>
      <Card style={{ width: '100%', maxWidth: '400px' }} className="mx-3">
        <Card.Body className="p-4">
          <h3 className="text-center mb-4">Вход в систему</h3>
          
          {error && <Alert variant="danger">{error}</Alert>}
          
          <Form onSubmit={handleSubmit}>
            <Form.Group className="mb-3">
              <Form.Label>Электронная почта</Form.Label>
              <Form.Control
                type="email"
                placeholder="Введите email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                size="lg"
              />
            </Form.Group>

            <Form.Group className="mb-3">
              <Form.Label>Пароль</Form.Label>
              <Form.Control
                type="password"
                placeholder="Введите пароль"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                size="lg"
              />
            </Form.Group>

            <Button 
              variant="primary" 
              type="submit" 
              className="w-100 mb-3"
              disabled={loading}
              size="lg"
            >
              {loading ? 'Вход...' : 'Войти'}
            </Button>
          </Form>

          <hr />
          
          <div className="mb-2">
            <small className="text-muted">Быстрый вход (демо):</small>
          </div>
          
          <div className="d-grid gap-2">
            <Button 
              variant="outline-primary" 
              size="sm"
              onClick={() => quickLogin('patient')}
            >
              Войти как Пациент
            </Button>
            <Button 
              variant="outline-success" 
              size="sm"
              onClick={() => quickLogin('clinic')}
            >
              Войти как Клиника
            </Button>
            <Button 
              variant="outline-secondary" 
              size="sm"
              onClick={() => quickLogin('government')}
            >
              Войти как Регулятор
            </Button>
          </div>

          <div className="text-center mt-3">
            <small>
              Нет аккаунта? <Link to="/register">Регистрация</Link>
            </small>
          </div>
        </Card.Body>
      </Card>
    </div>
  );
};

export default Login;
