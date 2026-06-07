import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import App from './App.vue'
import './main.css'
import './styles/app.css'
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'

const defaultThemeColors = {
  light: '#1867c0',
  dark: '#74a7ff',
}

const vuetify = createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: 'light',
    themes: {
      light: {
        dark: false,
        colors: {
          primary: defaultThemeColors.light,
        },
      },
      dark: {
        dark: true,
        colors: {
          primary: defaultThemeColors.dark,
        },
      },
    },
  },
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: {
      mdi,
    },
  },
  defaults: {
    VCard: {
      rounded: 'lg',
    },
    VTextField: {
      variant: 'outlined',
      density: 'compact',
      hideDetails: 'auto',
    },
    VSelect: {
      variant: 'outlined',
      density: 'compact',
      hideDetails: 'auto',
    },
    VBtn: {
      rounded: 'lg',
    },
  },
})

createApp(App).use(vuetify).mount('#app')
