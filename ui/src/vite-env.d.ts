/// <reference types="vite/client" />

// custom env vars consumed by the app; see .env.example
interface ImportMetaEnv {
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
