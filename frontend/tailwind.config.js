/** @type {import('tailwindcss').Config} */
const channel = (name) => `rgb(var(--${name}) / <alpha-value>)`;

export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        paper: channel('paper'),
        sheet: channel('sheet'),
        'paper-deep': channel('paper-deep'),
        ink: channel('ink'),
        muted: channel('muted'),
        faint: channel('faint'),
        line: channel('line'),
        mark: channel('mark'),
        'mark-soft': channel('mark-soft'),
        moss: channel('moss'),
        'moss-soft': channel('moss-soft'),
        warn: channel('warn'),
        'warn-soft': channel('warn-soft'),
        codebg: channel('codebg'),
        codebar: channel('codebar'),
        codetext: channel('codetext'),
        codecomment: channel('codecomment'),
        codestring: channel('codestring'),
        codekeyword: channel('codekeyword'),
      },
      fontFamily: {
        sans: ['IBM Plex Sans', 'Helvetica Neue', 'Arial', 'sans-serif'],
        display: ['Fraunces', 'Iowan Old Style', 'Palatino', 'serif'],
        mono: ['IBM Plex Mono', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
};
