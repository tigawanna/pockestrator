# Pockestrator Frontend

This is the frontend for the Pockestrator application, built with Preact, TypeScript, Tailwind CSS, and DaisyUI.

## Development Setup

### Prerequisites

- Node.js (v16 or later)
- npm or yarn

### Installation

```bash
# Install dependencies
npm install
```

### Development

```bash
# Start development server
npm run dev
```

This will start a development server at http://localhost:5173 with hot module replacement.

### Building for Production

```bash
# Build for production
npm run build:prod
```

This will create a `dist` directory with the compiled assets.

## Project Structure

```
pb_public/
├── dist/               # Built files (generated)
├── src/
│   ├── components/     # Reusable UI components
│   ├── pages/          # Page components
│   ├── services/       # API services
│   ├── types/          # TypeScript type definitions
│   ├── app.tsx         # Main application component
│   ├── main.tsx        # Application entry point
│   └── styles.css      # Global styles with Tailwind
├── index.html          # HTML template
├── package.json        # Project dependencies and scripts
├── tailwind.config.js  # Tailwind CSS configuration
├── tsconfig.json       # TypeScript configuration
└── vite.config.ts      # Vite configuration
```

## Integration with Go Backend

The frontend is embedded into the Go binary using Go's embed feature. In development mode, the application serves files directly from the `pb_public` directory. In production, it serves the embedded files from the binary.

### Development Workflow

1. Run the Go backend with `go run main.go`
2. In a separate terminal, run `npm run dev` in the `pb_public` directory
3. Access the frontend at http://localhost:5173 for direct Vite development
4. Or access via the Go backend at http://localhost:8090 (or your configured port)

### Production Build

1. Build the frontend: `cd pb_public && npm run build:prod`
2. Build the Go binary: `go build -o pockestrator`
3. Run the binary: `./pockestrator`

The frontend will be embedded in the binary and served directly.

## Technologies Used

- [Preact](https://preactjs.com/) - A fast 3kB alternative to React with the same API
- [TypeScript](https://www.typescriptlang.org/) - JavaScript with syntax for types
- [Tailwind CSS](https://tailwindcss.com/) - A utility-first CSS framework
- [DaisyUI](https://daisyui.com/) - Component library for Tailwind CSS
- [Vite](https://vitejs.dev/) - Next generation frontend tooling