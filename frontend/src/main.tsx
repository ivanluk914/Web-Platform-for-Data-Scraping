import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { NextUIProvider } from '@nextui-org/react'

import App from './App.tsx'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <NextUIProvider>
      <main className="w-screen h-screen bg-white">
        <App />
      </main>
    </NextUIProvider>
  </StrictMode>,
);