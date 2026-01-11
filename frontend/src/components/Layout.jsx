import { Container } from 'react-bootstrap';
import Navbar from './Navbar';

const Layout = ({ children }) => {
  return (
    <>
      <Navbar />
      <Container>
        {children}
      </Container>
    </>
  );
};

export default Layout;
