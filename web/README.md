# Case outgoings UI preview

A React/Vite design exploration for the Go API in the repository root. It includes a table and expandable case cards. Users enter case IDs to fetch estimates from the API, then can remove those cases. The UI starts empty and persists only the added IDs in browser storage; all case fields and outgoing amounts shown come from API responses.

## Run

```sh
cd web
npm install
npm run dev
```

The current backend exposes per-case estimates only. Set `VITE_API_BASE_URL` if the Go API is not running at `http://localhost:8080`. The API must permit cross-origin requests from the Vite development server (or serve the UI from the same origin).
