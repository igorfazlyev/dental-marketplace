import { useState, useEffect } from 'react';
import { Card, Button, Table, Alert, ProgressBar, Badge, Form } from 'react-bootstrap';
import { patientAPI } from '../api/client';
import { useAuth } from '../context/AuthContext';

const PatientDashboard = () => {
  const { user } = useAuth();
  const [studies, setStudies] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [message, setMessage] = useState({ type: '', text: '' });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStudies();
  }, []);

  const loadStudies = async () => {
    try {
      const response = await patientAPI.getStudies();
      setStudies(response.data.studies || []);
    } catch (error) {
      setMessage({ 
        type: 'danger', 
        text: error.response?.data?.error || 'Не удалось загрузить исследования' 
      });
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    if (!file.name.endsWith('.dcm')) {
      setMessage({ type: 'warning', text: 'Пожалуйста, выберите файл DICOM (.dcm)' });
      return;
    }

    setUploading(true);
    setUploadProgress(0);
    setMessage({ type: '', text: '' });

    try {
      await patientAPI.uploadDICOM(file, (progress) => {
        setUploadProgress(progress);
      });

      setMessage({ type: 'success', text: 'Файл успешно загружен!' });
      loadStudies();
      e.target.value = '';
    } catch (error) {
      setMessage({ 
        type: 'danger', 
        text: error.response?.data?.error || 'Не удалось загрузить файл' 
      });
    } finally {
      setUploading(false);
      setUploadProgress(0);
    }
  };

  const getStatusBadge = (status) => {
    const variants = {
      uploaded: 'primary',
      processing: 'warning',
      analyzed: 'success',
      failed: 'danger',
    };
    const labels = {
      uploaded: 'Загружен',
      processing: 'Обработка',
      analyzed: 'Проанализирован',
      failed: 'Ошибка',
    };
    return <Badge bg={variants[status] || 'secondary'}>{labels[status] || status}</Badge>;
  };

  const formatFileSize = (bytes) => {
    if (bytes < 1024) return bytes + ' Б';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' КБ';
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' МБ';
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' ГБ';
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString('ru-RU');
  };

  return (
    <div className="pb-4">
      <h2 className="mb-4">Личный кабинет пациента</h2>
      
      <Card className="mb-4">
        <Card.Body>
          <Card.Title>Добро пожаловать, {user.first_name}!</Card.Title>
          <Card.Text className="text-muted">
            Загрузите снимки КТ зубов для получения рекомендаций по лечению на основе ИИ.
          </Card.Text>
        </Card.Body>
      </Card>

      {message.text && (
        <Alert variant={message.type} dismissible onClose={() => setMessage({ type: '', text: '' })}>
          {message.text}
        </Alert>
      )}

      <Card className="mb-4">
        <Card.Body>
          <Card.Title>Загрузить снимок КТ</Card.Title>
          
          <div className="mb-3">
            <Form.Control
              type="file"
              accept=".dcm"
              onChange={handleFileUpload}
              disabled={uploading}
              size="lg"
            />
            <Form.Text className="text-muted">
              Принимаются только файлы DICOM (.dcm)
            </Form.Text>
          </div>

          {uploading && (
            <div>
              <div className="d-flex justify-content-between mb-1">
                <small>Загрузка...</small>
                <small>{uploadProgress}%</small>
              </div>
              <ProgressBar now={uploadProgress} animated />
            </div>
          )}
        </Card.Body>
      </Card>

      <Card>
        <Card.Body>
          <Card.Title>Мои исследования ({studies.length})</Card.Title>
          
          {loading ? (
            <div className="text-center py-5">
              <div className="spinner-border text-primary" role="status">
                <span className="visually-hidden">Загрузка...</span>
              </div>
            </div>
          ) : studies.length === 0 ? (
            <Alert variant="info">
              Исследований пока нет. Загрузите ваш первый снимок КТ выше.
            </Alert>
          ) : (
            <div className="table-responsive">
              <Table striped bordered hover>
                <thead>
                  <tr>
                    <th className="d-none d-md-table-cell">#</th>
                    <th>Описание</th>
                    <th>Статус</th>
                    <th className="d-none d-lg-table-cell">Размер</th>
                    <th className="d-none d-md-table-cell">Загружено</th>
                    <th>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {studies.map((study) => (
                    <tr key={study.id}>
                      <td className="d-none d-md-table-cell">{study.id}</td>
                      <td>{study.description || 'Без названия'}</td>
                      <td>{getStatusBadge(study.status)}</td>
                      <td className="d-none d-lg-table-cell">{formatFileSize(study.file_size)}</td>
                      <td className="d-none d-md-table-cell">{formatDate(study.created_at)}</td>
                      <td>
                        <Button 
                          variant="outline-primary" 
                          size="sm"
                          onClick={() => window.open(`http://localhost:8042/app/explorer.html#study?uuid=${study.orthanc_study_id}`, '_blank')}
                        >
                          <span className="d-none d-md-inline">Просмотр</span>
                          <span className="d-md-none">👁️</span>
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            </div>
          )}
        </Card.Body>
      </Card>
    </div>
  );
};

export default PatientDashboard;
