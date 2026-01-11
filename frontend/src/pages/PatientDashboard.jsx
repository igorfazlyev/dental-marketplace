import { useState, useEffect } from 'react';
import { Card, Button, Table, Alert, ProgressBar, Badge, Form, Spinner, Modal, Accordion } from 'react-bootstrap';
import { patientAPI, diagnocatAPI } from '../api/client';
import { useAuth } from '../context/AuthContext';


const PatientDashboard = () => {
  const { user } = useAuth();
  const [studies, setStudies] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [message, setMessage] = useState({ type: '', text: '' });
  const [loading, setLoading] = useState(true);
  const [sendingToDiagnocat, setSendingToDiagnocat] = useState({});
  const [diagnocatAnalyses, setDiagnocatAnalyses] = useState([]);
  const [showAnalysisModal, setShowAnalysisModal] = useState(false);
  const [selectedAnalysis, setSelectedAnalysis] = useState(null);
  const [refreshingAnalysis, setRefreshingAnalysis] = useState({});
  const [uploadDestination, setUploadDestination] = useState('diagnocat'); // NEW STATE


  useEffect(() => {
    loadStudies();
    loadDiagnocatAnalyses();
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


  const loadDiagnocatAnalyses = async () => {
    try {
      const response = await diagnocatAPI.getAnalyses();
      setDiagnocatAnalyses(response.data.analyses || []);
    } catch (error) {
      console.error('Failed to load Diagnocat analyses:', error);
    }
  };


  // UPDATED UPLOAD HANDLER
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
      // Create FormData with file and destination
      const formData = new FormData();
      formData.append('file', file);
      formData.append('destination', uploadDestination); // Add destination parameter

      await patientAPI.uploadDICOM(formData, (progress) => {
        setUploadProgress(progress);
      });

      // Show appropriate success message
      if (uploadDestination === 'diagnocat') {
        setMessage({ 
          type: 'success', 
          text: '✅ Файл загружен в Diagnocat! AI-анализ начался автоматически. Результаты появятся через несколько минут.' 
        });
      } else {
        setMessage({ 
          type: 'success', 
          text: '✅ Файл сохранен в локальном хранилище Orthanc.' 
        });
      }

      loadStudies();
      loadDiagnocatAnalyses();
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


  const handleSendToDiagnocat = async (studyId) => {
    setSendingToDiagnocat(prev => ({ ...prev, [studyId]: true }));
    setMessage({ type: '', text: '' });

    try {
      await diagnocatAPI.sendStudy(studyId);
      setMessage({ 
        type: 'success', 
        text: 'Исследование успешно отправлено в Diagnocat! Анализ может занять несколько минут.' 
      });
      loadStudies();
      loadDiagnocatAnalyses();
    } catch (error) {
      setMessage({ 
        type: 'danger', 
        text: error.response?.data?.error || 'Не удалось отправить в Diagnocat. Попробуйте позже.' 
      });
    } finally {
      setSendingToDiagnocat(prev => ({ ...prev, [studyId]: false }));
    }
  };


  const handleRefreshAnalysis = async (analysisId) => {
    setRefreshingAnalysis(prev => ({ ...prev, [analysisId]: true }));

    try {
      const response = await diagnocatAPI.refreshAnalysis(analysisId);
      setMessage({ type: 'success', text: 'Статус анализа обновлен!' });
      loadDiagnocatAnalyses();
      
      if (selectedAnalysis?.id === analysisId) {
        setSelectedAnalysis(response.data.analysis);
      }
    } catch (error) {
      setMessage({ 
        type: 'danger', 
        text: error.response?.data?.error || 'Не удалось обновить статус анализа' 
      });
    } finally {
      setRefreshingAnalysis(prev => ({ ...prev, [analysisId]: false }));
    }
  };


  const handleViewAnalysis = (studyId) => {
    const analysis = diagnocatAnalyses.find(a => a.study_id === studyId);
    if (analysis) {
      setSelectedAnalysis(analysis);
      setShowAnalysisModal(true);
    }
  };


  const getDiagnocatAnalysisForStudy = (studyId) => {
    return diagnocatAnalyses.find(a => a.study_id === studyId);
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


  const getDiagnocatStatusBadge = (status) => {
    const variants = {
      uploading: 'info',
      processing: 'warning',
      complete: 'success',
      failed: 'danger',
    };
    const labels = {
      uploading: 'Загрузка',
      processing: 'Анализ',
      complete: 'Готово',
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


  const getAttributeName = (attributeId) => {
    const attributeMap = {
      1: 'Кариес',
      2: 'Пульпит',
      3: 'Периодонтит',
      4: 'Пломба',
      5: 'Коронка',
      6: 'Имплант',
      7: 'Перелом корня',
      8: 'Резорбция корня',
      9: 'Киста',
      10: 'Гранулема',
    };
    return attributeMap[attributeId] || `Признак #${attributeId}`;
  };


  const getTotalPathologies = (diagnoses) => {
    if (!diagnoses?.diagnoses) return 0;
    return diagnoses.diagnoses.reduce((total, d) => total + d.attributes.length, 0);
  };


  const getAffectedTeethCount = (diagnoses) => {
    if (!diagnoses?.diagnoses) return 0;
    return diagnoses.diagnoses.length;
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

      {/* UPDATED UPLOAD CARD */}
      <Card className="mb-4">
        <Card.Body>
          <Card.Title>Загрузить снимок КТ</Card.Title>
          
          {/* NEW: Destination Selector */}
          <Form.Group className="mb-3">
            <Form.Label>Куда загрузить снимок?</Form.Label>
            <div>
              <Form.Check
                inline
                type="radio"
                label="🤖 Diagnocat AI (рекомендуется - анализ сразу)"
                name="destination"
                id="dest-diagnocat"
                checked={uploadDestination === 'diagnocat'}
                onChange={() => setUploadDestination('diagnocat')}
                disabled={uploading}
              />
              <Form.Check
                inline
                type="radio"
                label="💾 Локальное хранилище Orthanc"
                name="destination"
                id="dest-orthanc"
                checked={uploadDestination === 'orthanc'}
                onChange={() => setUploadDestination('orthanc')}
                disabled={uploading}
              />
            </div>
            <Form.Text className="text-muted">
              {uploadDestination === 'diagnocat' 
                ? '✅ Файл загрузится в облако Diagnocat для немедленного AI-анализа зубов'
                : '📁 Файл сохранится локально в системе Orthanc без анализа (можно отправить на анализ позже)'}
            </Form.Text>
          </Form.Group>
          
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
                <small>
                  {uploadDestination === 'diagnocat' 
                    ? '🚀 Загрузка в Diagnocat AI...' 
                    : '📤 Загрузка в Orthanc...'}
                </small>
                <small>{uploadProgress}%</small>
              </div>
              <ProgressBar 
                now={uploadProgress} 
                animated 
                variant={uploadDestination === 'diagnocat' ? 'success' : 'primary'}
              />
            </div>
          )}
        </Card.Body>
      </Card>

      {/* Rest of the component stays the same */}
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
                  {studies.map((study) => {
                    const diagnocatAnalysis = getDiagnocatAnalysisForStudy(study.id);
                    const hasOrthancId = study.orthanc_study_id; // Check if stored in Orthanc
                    
                    return (
                      <tr key={study.id}>
                        <td className="d-none d-md-table-cell">{study.id}</td>
                        <td>
                          <div>{study.description || 'Без названия'}</div>
                          {diagnocatAnalysis && (
                            <div className="mt-1">
                              <small className="text-muted d-flex align-items-center gap-1">
                                🤖 Diagnocat: {getDiagnocatStatusBadge(diagnocatAnalysis.status)}
                                {diagnocatAnalysis.complete && diagnocatAnalysis.diagnoses?.diagnoses && (
                                  <Badge bg="warning" className="ms-2">
                                    {getAffectedTeethCount(diagnocatAnalysis.diagnoses)} зубов с проблемами
                                  </Badge>
                                )}
                              </small>
                            </div>
                          )}
                        </td>
                        <td>{getStatusBadge(study.status)}</td>
                        <td className="d-none d-lg-table-cell">{formatFileSize(study.file_size)}</td>
                        <td className="d-none d-md-table-cell">{formatDate(study.created_at)}</td>
                        <td>
                          <div className="d-flex flex-column flex-md-row gap-2">
                            {/* Show Orthanc viewer button only if stored in Orthanc */}
                            {hasOrthancId && (
                              <Button 
                                variant="outline-primary" 
                                size="sm"
                                onClick={() => window.open(`http://localhost:8042/app/explorer.html#study?uuid=${study.orthanc_study_id}`, '_blank')}
                              >
                                <span className="d-none d-md-inline">👁️ Orthanc</span>
                                <span className="d-md-none">👁️</span>
                              </Button>
                            )}

                            {diagnocatAnalysis ? (
                              diagnocatAnalysis.complete ? (
                                <Button
                                  variant="success"
                                  size="sm"
                                  onClick={() => handleViewAnalysis(study.id)}
                                >
                                  <span className="d-none d-md-inline">📄 Результаты ИИ</span>
                                  <span className="d-md-none">📄</span>
                                </Button>
                              ) : (
                                <Button
                                  variant="outline-secondary"
                                  size="sm"
                                  onClick={() => handleRefreshAnalysis(diagnocatAnalysis.id)}
                                  disabled={refreshingAnalysis[diagnocatAnalysis.id]}
                                >
                                  {refreshingAnalysis[diagnocatAnalysis.id] ? (
                                    <Spinner animation="border" size="sm" />
                                  ) : (
                                    <>
                                      <span className="d-none d-md-inline">🔄 Обновить</span>
                                      <span className="d-md-none">🔄</span>
                                    </>
                                  )}
                                </Button>
                              )
                            ) : hasOrthancId ? (
                              // Show "Send to AI" button only for Orthanc-stored studies
                              <Button 
                                variant="info" 
                                size="sm"
                                onClick={() => handleSendToDiagnocat(study.id)}
                                disabled={sendingToDiagnocat[study.id]}
                              >
                                {sendingToDiagnocat[study.id] ? (
                                  <Spinner animation="border" size="sm" />
                                ) : (
                                  <>
                                    <span className="d-none d-md-inline">🤖 Анализ ИИ</span>
                                    <span className="d-md-none">🤖</span>
                                  </>
                                )}
                              </Button>
                            ) : null}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </Table>
            </div>
          )}
        </Card.Body>
      </Card>

      {/* Modal stays the same - keeping your existing detailed analysis modal */}
      <Modal 
        show={showAnalysisModal} 
        onHide={() => setShowAnalysisModal(false)}
        size="xl"
        scrollable
      >
        <Modal.Header closeButton>
          <Modal.Title>Результаты анализа Diagnocat AI</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {selectedAnalysis ? (
            <div>
              <Card className="mb-3">
                <Card.Body>
                  <div className="row">
                    <div className="col-md-4 mb-2">
                      <strong>Тип анализа:</strong><br />
                      <Badge bg="info">{selectedAnalysis.analysis_type || 'Не указан'}</Badge>
                    </div>
                    <div className="col-md-4 mb-2">
                      <strong>Статус:</strong><br />
                      {getDiagnocatStatusBadge(selectedAnalysis.status)}
                    </div>
                    <div className="col-md-4 mb-2">
                      <strong>Дата создания:</strong><br />
                      {formatDate(selectedAnalysis.created_at)}
                    </div>
                  </div>
                </Card.Body>
              </Card>
              
              {selectedAnalysis.error && (
                <Alert variant="danger">
                  <strong>Ошибка:</strong> {selectedAnalysis.error}
                </Alert>
              )}

              {selectedAnalysis.complete && selectedAnalysis.diagnoses?.diagnoses && (
                <>
                  <Alert variant="warning" className="mb-3">
                    <h5 className="mb-3">📊 Сводка результатов</h5>
                    <div className="row text-center">
                      <div className="col-6 col-md-3">
                        <h3 className="text-danger">{getAffectedTeethCount(selectedAnalysis.diagnoses)}</h3>
                        <small>Зубов с проблемами</small>
                      </div>
                      <div className="col-6 col-md-3">
                        <h3 className="text-warning">{getTotalPathologies(selectedAnalysis.diagnoses)}</h3>
                        <small>Всего патологий</small>
                      </div>
                      <div className="col-6 col-md-3">
                        <h3 className="text-info">
                          {selectedAnalysis.diagnoses.diagnoses.filter(d => d.periodontal_status?.roots?.length > 0).length}
                        </h3>
                        <small>С пародонтальными данными</small>
                      </div>
                      <div className="col-6 col-md-3">
                        <h3 className="text-success">
                          {selectedAnalysis.diagnoses.diagnoses.filter(d => d.text_comment).length}
                        </h3>
                        <small>С комментариями</small>
                      </div>
                    </div>
                  </Alert>

                  <Card className="mb-3">
                    <Card.Header>
                      <h5 className="mb-0">🦷 Детальная диагностика по зубам</h5>
                    </Card.Header>
                    <Card.Body style={{ maxHeight: '400px', overflowY: 'auto' }}>
                      <Accordion>
                        {selectedAnalysis.diagnoses.diagnoses.map((diagnosis, idx) => (
                          <Accordion.Item eventKey={idx.toString()} key={idx}>
                            <Accordion.Header>
                              <div className="d-flex justify-content-between align-items-center w-100 me-3">
                                <strong>Зуб #{diagnosis.tooth_number}</strong>
                                <div className="d-flex gap-2">
                                  {diagnosis.attributes.length > 0 && (
                                    <Badge bg="danger">{diagnosis.attributes.length} проблем(ы)</Badge>
                                  )}
                                  {diagnosis.periodontal_status?.roots?.length > 0 && (
                                    <Badge bg="info">Пародонт</Badge>
                                  )}
                                  {diagnosis.text_comment && (
                                    <Badge bg="secondary">Комментарий</Badge>
                                  )}
                                </div>
                              </div>
                            </Accordion.Header>
                            <Accordion.Body>
                              {diagnosis.attributes.length > 0 && (
                                <div className="mb-3">
                                  <strong>Обнаруженные патологии:</strong>
                                  <Table size="sm" className="mt-2">
                                    <thead>
                                      <tr>
                                        <th>Патология</th>
                                        <th className="text-center">AI</th>
                                        <th className="text-center">Врач</th>
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {diagnosis.attributes.map((attr, i) => (
                                        <tr key={i}>
                                          <td>{getAttributeName(attr.attribute_id)}</td>
                                          <td className="text-center">
                                            {attr.model_positive ? '✅' : '❌'}
                                          </td>
                                          <td className="text-center">
                                            {attr.user_decision ? (attr.user_positive ? '✅' : '❌') : '⏳'}
                                          </td>
                                        </tr>
                                      ))}
                                    </tbody>
                                  </Table>
                                </div>
                              )}

                              {diagnosis.periodontal_status?.roots?.length > 0 && (
                                <div className="mb-3">
                                  <strong>Пародонтальные измерения:</strong>
                                  <div className="mt-2">
                                    {diagnosis.periodontal_status.roots.map((root, i) => (
                                      <div key={i} className="mb-2">
                                        <Badge bg="info">{root.root || 'Корень'}</Badge>
                                        {root.measurements?.length && (
                                          <span className="ms-2">
                                            Длина: {root.measurements.length.predicted?.toFixed(2) || 'N/A'} мм
                                          </span>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}

                              {diagnosis.text_comment && (
                                <Alert variant="secondary" className="mb-0">
                                  <strong>Комментарий:</strong> {diagnosis.text_comment}
                                </Alert>
                              )}
                            </Accordion.Body>
                          </Accordion.Item>
                        ))}
                      </Accordion>
                    </Card.Body>
                  </Card>
                </>
              )}

              {selectedAnalysis.complete && (
                <Card>
                  <Card.Header>
                    <h5 className="mb-0">📥 Скачать отчеты</h5>
                  </Card.Header>
                  <Card.Body>
                    <div className="d-grid gap-2">
                      {selectedAnalysis.pdf_url && (
                        <Button 
                          variant="primary" 
                          onClick={() => window.open(selectedAnalysis.pdf_url, '_blank')}
                        >
                          📄 Скачать PDF отчет
                        </Button>
                      )}
                      
                      {selectedAnalysis.webpage_url && (
                        <Button 
                          variant="outline-primary" 
                          onClick={() => window.open(selectedAnalysis.webpage_url, '_blank')}
                        >
                          🌐 Открыть интерактивный веб-отчет
                        </Button>
                      )}
                      
                      {selectedAnalysis.preview_url && (
                        <Button 
                          variant="outline-secondary" 
                          onClick={() => window.open(selectedAnalysis.preview_url, '_blank')}
                        >
                          👁️ Посмотреть превью
                        </Button>
                      )}
                    </div>
                  </Card.Body>
                </Card>
              )}

              {!selectedAnalysis.complete && (
                <Alert variant="info">
                  ⏳ Анализ все еще обрабатывается. Пожалуйста, вернитесь позже или нажмите кнопку "Обновить" в таблице.
                </Alert>
              )}
            </div>
          ) : (
            <p>Анализ не найден</p>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowAnalysisModal(false)}>
            Закрыть
          </Button>
          {selectedAnalysis && !selectedAnalysis.complete && (
            <Button 
              variant="primary" 
              onClick={() => handleRefreshAnalysis(selectedAnalysis.id)}
              disabled={refreshingAnalysis[selectedAnalysis.id]}
            >
              {refreshingAnalysis[selectedAnalysis.id] ? (
                <>
                  <Spinner animation="border" size="sm" className="me-2" />
                  Обновление...
                </>
              ) : (
                '🔄 Обновить статус'
              )}
            </Button>
          )}
        </Modal.Footer>
      </Modal>
    </div>
  );
};

export default PatientDashboard;
