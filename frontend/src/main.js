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
import 'vidstack/styles/base.css'
import 'vidstack/styles/defaults.css'
import 'vidstack/styles/community-skin/video.css'
import 'vidstack/define/media-player.js'
import 'vidstack/define/media-outlet.js'
import 'vidstack/define/media-community-skin.js'

const vuetify = createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: 'light',
    themes: {
      light: {
        dark: false,
        colors: {
          primary: '#1867c0',
        },
      },
      dark: {
        dark: true,
        colors: {
          primary: '#74a7ff',
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
