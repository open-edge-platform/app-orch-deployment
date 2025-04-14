// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

const expectedParams = ['project', 'cluster', 'namespace', 'service']

class CookieError extends Error {
  constructor (message, existingQuery, newQuery) {
    super(message)
    this.existingQuery = existingQuery
    this.newQuery = newQuery
  }
}

function existingQuery () {
  const existingQp = {}
  expectedParams.forEach((param) => {
    const cookieName = `app-service-proxy-${param}`
    const existingCookie = document.cookie.split('; ').find(row => row.startsWith(cookieName))
    if (existingCookie) {
      const existingCookieValue = existingCookie.split('=')[1]
      existingQp[param] = existingCookieValue
    }
  })
  return existingQp
}

function newQuery () {
  const newQp = {}
  const queryParams = new URLSearchParams(window.location.search)
  expectedParams.forEach((param) => {
    if (queryParams.has(param)) {
      const value = queryParams.get(param)
      newQp[param] = value
    }
  })
  return newQp
}

function deleteAllCookies () {
  const cookies = document.cookie.split(';')
  for (const cookie of cookies) {
    const cookieName = cookie.split('=')[0].trim()
    document.cookie = `${cookieName}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 UTC; SameSite=Strict`
  }
  console.log('All cookies have been deleted.')
}

document.addEventListener('DOMContentLoaded', function () {
  const Keycloak = window.Keycloak

  // Initialize Keycloak
  const domain = window.location.hostname.split('.').slice(1).join('.')
  let keycloakUrl = 'https://keycloak.' + domain
  if (domain === '') {
    // For local development - see README.md
    keycloakUrl = 'http://localhost:8090'
  }
  const keycloak = new Keycloak({
    url: keycloakUrl,
    realm: 'master',
    clientId: 'webui-client'
  })

  keycloak.onAuthLogout = function () {
    console.log('Logout event triggered. Deleting cookies')
    deleteAllCookies()
  }

  keycloak.init({
    onLoad: 'login-required'
  }).then(function (authenticated) {
    console.log(authenticated ? 'User is authenticated' : 'User is not authenticated')

    // Check for required query parameters and save them as cookies
    const queryParams = new URLSearchParams(window.location.search)
    expectedParams.forEach((param) => {
      if (!queryParams.has(param)) {
        console.log(`Query parameter missing: ${param}`)
        throw new Error(`Query parameter missing: ${param}`)
      }
      const value = queryParams.get(param)
      const cookieName = `app-service-proxy-${param}`
      const existingCookie = document.cookie.split('; ').find(row => row.startsWith(cookieName))
      if (!existingCookie) {
        console.log(`Query parameter added as cookie: ${cookieName} = ${value}`)
        document.cookie = `${cookieName}=${value}; path=/; SameSite=Strict`
        return
      }
      const existingCookieValue = existingCookie.split('=')[1]
      if (existingCookieValue !== value) {
        throw new CookieError('Changing context from/to', existingQuery(), newQuery())
      }
    })

    // Set the token as a cookie. If it is longer than 2k break it up into multiple cookies
    const token = keycloak.token
    const maxCookieLength = 2000
    const numCookies = Math.ceil(token.length / maxCookieLength)
    for (let i = 0; i < numCookies; i++) {
      const cookieValue = token.substring(i * maxCookieLength, (i + 1) * maxCookieLength)
      document.cookie = `app-service-proxy-token-${i}=${cookieValue}; path=/; secure; SameSite=Strict`
    }
    document.cookie = `app-service-proxy-tokens=${numCookies}; path=/; SameSite=Strict`
    console.log(`Token saved as ${numCookies} cookies`)

    // Set the token expiration time as a cookie
    const expirationTime = keycloak.tokenParsed.exp * 1000
    const expirationDate = new Date(expirationTime)
    document.cookie = `app-service-proxy-token-expiration=${expirationDate.toUTCString()}; path=/; SameSite=Strict`

    // Redirect to the application
    const windowUrl = window.location.protocol + '//' + window.location.hostname + ':' + window.location.port
    console.log('Loading iFrame. URL: ' + windowUrl)

    // To facilitate local development, load the CSS file from the same domain as the iframe
    if (!windowUrl.includes('localhost')) {
      document.getElementById('asp-iframe').src = windowUrl + '/'
    } else {
      document.getElementById('asp-iframe').src = windowUrl + '/app-service-proxy-styles.css'
    }
  }).catch(function (err) {
    if (err instanceof CookieError) {
      console.error('Only one App Service Proxy can be open at a time:', err.message, err.newQuery)
      const dialog = document.getElementById('app-service-proxy-dialog')
      dialog.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
          event.preventDefault()
        }
      })

      expectedParams.forEach((param) => {
        document.getElementById(`new-${param}-id`).innerText = err.newQuery[param]
        document.getElementById(`old-${param}-id`).innerText = err.existingQuery[param]
        if (err.existingQuery[param] !== err.newQuery[param]) {
          document.getElementById(`old-${param}-id`).style.color = 'green'
          document.getElementById(`new-${param}-id`).style.color = 'darkgreen'
        } else {
          document.getElementById(`old-${param}`).hidden = true
          document.getElementById(`new-${param}`).hidden = true
        }
      })

      const changeButton = document.getElementById('change-button')

      dialog.showModal()

      changeButton.addEventListener('click', () => {
        deleteAllCookies()
        window.location.reload() // Reload the page to reinitialize ASP and Keycloak
      })
    } else if (err instanceof Object && err.error !== undefined) {
      const dialogErr = document.getElementById('app-service-proxy-error-dialog')
      dialogErr.showModal()
      const errorMessage = document.getElementById('error-message')
      errorMessage.innerText = `Keycloak unavailable at: ${keycloakUrl}. \nError ${err.error}`
    } else {
      const dialogErr = document.getElementById('app-service-proxy-error-dialog')
      dialogErr.showModal()
      const errorMessage = document.getElementById('error-message')
      errorMessage.innerText = `Error ${err.message}`
    }
  })
})
