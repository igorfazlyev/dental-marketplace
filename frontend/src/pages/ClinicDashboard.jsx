import { Card } from 'react-bootstrap';
import { useAuth } from '../context/AuthContext';

const ClinicDashboard = () => {
  const { user } = useAuth();

  return (
    <div>
      <h2 className="mb-4">Панель управления клиники</h2>
      
      <Card>
        <Card.Body>
          <Card.Title>Добро пожаловать, {user.first_name}!</Card.Title>
          <Card.Text className="text-muted">
            Функционал для клиник в разработке...
          </Card.Text>
        </Card.Body>
      </Card>
    </div>
  );
};

export default ClinicDashboard;
