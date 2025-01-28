import { Link, useLocation } from "react-router-dom";

const Navigation = () => {
  const location = useLocation();
  const isActive = (path: string) => location.pathname === path;

  return (
    <nav className="bg-white border-b border-gray-200 dark:bg-gray-800 dark:border-gray-700">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center h-16">
          <div className="flex items-center space-x-4">
            <img
              src="https://github.com/grafana/loki/blob/main/docs/sources/logo.png?raw=true"
              alt="Grafana Loki Logo"
              className="h-8 w-8"
            />
            <h1 className="text-xl text-gray-900 dark:text-white font-bold">
              Compactor
            </h1>
            <div className="ml-8">
              <div className="flex items-center space-x-4">
                <Link
                  to="/"
                  className={`${
                    isActive("/")
                      ? "bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-white"
                      : "text-gray-500 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-white"
                  } px-3 py-2 rounded-md text-sm font-medium transition-colors`}
                >
                  Ring
                </Link>
                <Link
                  to="/deletes"
                  className={`${
                    isActive("/deletes")
                      ? "bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-white"
                      : "text-gray-500 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-white"
                  } px-3 py-2 rounded-md text-sm font-medium transition-colors`}
                >
                  Deletes
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navigation;
