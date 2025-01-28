import { Routes, Route } from "react-router-dom";
import Navigation from "./components/Navigation";
import Ring from "./pages/Ring";
import Deletes from "./pages/Deletes";
import Layout from "./components/Layout";
import NewDeleteRequest from "./pages/NewDeleteRequest";

const App = () => {
  return (
    <Layout>
      <Navigation />
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <Routes>
          <Route path="/" element={<Ring />} />
          <Route path="/deletes" element={<Deletes />} />
          <Route path="/deletes/new" element={<NewDeleteRequest />} />
        </Routes>
      </div>
    </Layout>
  );
};

export default App;
