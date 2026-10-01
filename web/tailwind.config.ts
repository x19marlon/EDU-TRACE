import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-nunito)", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["'JetBrains Mono'", "'Fira Code'", "'Cascadia Code'", "ui-monospace", "monospace"],
      },
      colors: {
        cream: "#FBF7F2",
        ink: {
          DEFAULT: "#2E2A3F",
          soft: "#6B6680",
          faint: "#A29DB3",
        },
        lavender: {
          50: "#F6F3FE",
          100: "#EDE7FC",
          200: "#DCD2F8",
          300: "#C3B4F1",
          400: "#A38DE6",
          500: "#8A6FDD",
          600: "#7154CC",
          700: "#5C42AE",
        },
        mint: {
          50: "#F0FAF5",
          100: "#DDF3EA",
          200: "#BFE6D6",
          300: "#9AD6BE",
          400: "#6FC2A2",
          600: "#3A8F6F",
          700: "#2D7258",
        },
        peach: {
          50: "#FFF6EF",
          100: "#FDEBDD",
          200: "#F9D8C0",
          300: "#F5BE9A",
          400: "#EFA072",
          600: "#C9683A",
          700: "#A4522C",
        },
      },
      boxShadow: {
        soft: "0 1px 2px rgb(46 42 63 / 0.04), 0 8px 24px -8px rgb(46 42 63 / 0.12)",
        lift: "0 2px 4px rgb(46 42 63 / 0.05), 0 16px 40px -12px rgb(46 42 63 / 0.22)",
      },
    },
  },
  plugins: [],
};
export default config;
