import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider, createBrowserRouter } from "react-router-dom";
import { BasenameProvider } from "./contexts/BasenameContext";

import App from "./App";
import "./index.css";

// Extract basename from current URL by matching everything up to and including /dataobj/explorer
const pathname = window.location.pathname;
const match = pathname.match(/(.*\/compactor\/ui\/)/);
const basename = match?.[1] || "/compactor/ui/";

const router = createBrowserRouter(
  [
    {
      path: "*",
      element: <App />,
    },
  ],
  {
    basename,
    future: {
      v7_relativeSplatPath: true,
    },
  }
);

const root = document.getElementById("root");
if (!root) throw new Error("Root element not found");

createRoot(root).render(
  <StrictMode>
    <BasenameProvider basename={basename}>
      <RouterProvider
        router={router}
        future={{
          v7_startTransition: true,
        }}
      />
    </BasenameProvider>
  </StrictMode>
);
