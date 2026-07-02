// 应用主题到 CSS 变量
function applyTheme(themeKey) {
    const colors = {
        light: {
            bgPage: '#f0f2f5', bgCard: '#ffffff', bgCardAlt: '#f8f9fa',
            bgSidebar: '#001529', bgHeader: '#ffffff', bgHover: '#f5f7fa',
            primary: '#1677ff', primaryLight: '#e6f4ff', primaryDark: '#0958d9',
            textPrimary: '#1d2129', textRegular: '#4e5969', textSecondary: '#86909c',
            border: '#e5e6eb', borderLight: '#f0f0f0',
            success: '#52c41a', warning: '#fa8c16', danger: '#f5222d', info: '#1677ff',
            menuText: 'rgba(255,255,255,0.85)', menuActive: '#ffffff',
        },
        dark: {
            bgPage: '#0f1419', bgCard: '#1e2937', bgCardAlt: '#243040',
            bgSidebar: '#0d1b2a', bgHeader: '#1a2332', bgHover: '#2a3a4e',
            primary: '#36a3ff', primaryLight: 'rgba(54,163,255,0.12)', primaryDark: '#1677ff',
            textPrimary: '#f5f7fa', textRegular: '#c9cdd4', textSecondary: '#86909c',
            border: '#333f4e', borderLight: '#2a3a4e',
            success: '#52c41a', warning: '#fa8c16', danger: '#f5222d', info: '#36a3ff',
            menuText: 'rgba(255,255,255,0.75)', menuActive: '#ffffff',
        }
    }
    const theme = colors[themeKey] || colors.light
    const root = document.documentElement
    Object.entries(theme).forEach(([key, value]) => {
        root.style.setProperty('--' + key.replace(/([A-Z])/g, '-$1').toLowerCase(), value)
    })
    root.setAttribute('data-theme', themeKey)
    localStorage.setItem('app-theme', themeKey)
}

export function initTheme() {
    applyTheme(localStorage.getItem('app-theme') || 'light')
}

export function setTheme(key) {
    applyTheme(key)
}
