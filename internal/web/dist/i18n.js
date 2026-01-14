// Simple i18n implementation for SuperTunnel
class I18n {
    constructor() {
        this.locale = this.detectLocale();
        this.translations = {};
        this.fallbackLocale = 'en';
    }

    detectLocale() {
        // Check localStorage first
        const saved = localStorage.getItem('locale');
        if (saved) return saved;

        // Detect from browser
        const browserLang = navigator.language || navigator.userLanguage;
        const lang = browserLang.split('-')[0]; // Get 'zh' from 'zh-CN'

        // Support only en, zh, ja
        if (['en', 'zh', 'ja'].includes(lang)) {
            return lang;
        }

        return 'en';
    }

    async load() {
        try {
            const response = await fetch(`/locales/${this.locale}.json`);
            this.translations = await response.json();
        } catch (err) {
            console.error(`Failed to load locale ${this.locale}:`, err);
            // Try fallback
            if (this.locale !== this.fallbackLocale) {
                const response = await fetch(`/locales/${this.fallbackLocale}.json`);
                this.translations = await response.json();
            }
        }
    }

    t(key, params = {}) {
        let text = this.translations[key] || key;

        // Simple template replacement
        Object.keys(params).forEach(param => {
            text = text.replace(`{{.${param}}}`, params[param]);
        });

        return text;
    }

    setLocale(locale) {
        if (['en', 'zh', 'ja'].includes(locale)) {
            this.locale = locale;
            localStorage.setItem('locale', locale);
            return this.load();
        }
    }

    getLocale() {
        return this.locale;
    }

    getSupportedLocales() {
        return [
            { code: 'en', name: 'English' },
            { code: 'zh', name: '简体中文' },
            { code: 'ja', name: '日本語' }
        ];
    }
}

// Global instance
const i18n = new I18n();
