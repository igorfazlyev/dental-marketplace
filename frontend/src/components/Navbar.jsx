import { Link, useNavigate } from 'react-router-dom';
import { Navbar as BootstrapNavbar, Container, Nav, Button } from 'react-bootstrap';
import { useAuth } from '../context/AuthContext';

const Navbar = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const getRoleLabel = (role) => {
    const labels = {
      patient: 'Пациент',
      clinic: 'Клиника',
      government: 'Регулятор',
    };
    return labels[role] || role;
  };

  return (
    <BootstrapNavbar bg="primary" variant="dark" expand="lg" className="mb-3 mb-md-4">
      <Container>
        <BootstrapNavbar.Brand as={Link} to="/" className="fw-bold">
          <span className="d-none d-md-inline">🦷 Стоматологическая Площадка</span>
          <span className="d-md-none">🦷 Стомат.</span>
        </BootstrapNavbar.Brand>
        
        <BootstrapNavbar.Toggle aria-controls="basic-navbar-nav" />
        
        <BootstrapNavbar.Collapse id="basic-navbar-nav" className="justify-content-end">
          {user ? (
            <Nav className="align-items-lg-center">
              <Nav.Link as={Link} to={`/${user.role}`} className="me-2">
                Панель управления
              </Nav.Link>
              <Nav.Item className="d-none d-lg-flex align-items-center mx-3 text-white">
                {user.first_name} {user.last_name} ({getRoleLabel(user.role)})
              </Nav.Item>
              <Nav.Item className="d-lg-none text-white-50 small px-3 py-1">
                {user.first_name} {user.last_name}
              </Nav.Item>
              <Button variant="outline-light" size="sm" onClick={handleLogout} className="mt-2 mt-lg-0">
                Выход
              </Button>
            </Nav>
          ) : (
            <Nav>
              <Nav.Link as={Link} to="/login">Вход</Nav.Link>
              <Nav.Link as={Link} to="/register">Регистрация</Nav.Link>
            </Nav>
          )}
        </BootstrapNavbar.Collapse>
      </Container>
    </BootstrapNavbar>
  );
};

export default Navbar;
