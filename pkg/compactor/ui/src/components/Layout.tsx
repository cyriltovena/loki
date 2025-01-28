import { PropsWithChildren, useEffect, useState } from "react";
import { DarkModeToggle } from "./DarkModeToggle";

const Layout: React.FC<PropsWithChildren> = ({ children }) => {
  const [isDarkMode, setIsDarkMode] = useState(() => {
    if (typeof window === "undefined") return false;
    return document.documentElement.classList.contains("dark");
  });

  useEffect(() => {
    if (isDarkMode) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  }, [isDarkMode]);

  const toggleDarkMode = () => {
    setIsDarkMode(!isDarkMode);
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="absolute top-3 right-4">
        <DarkModeToggle isDarkMode={isDarkMode} onToggle={toggleDarkMode} />
      </div>
      <main className="min-h-screen">{children}</main>
    </div>
  );
};

export default Layout;
