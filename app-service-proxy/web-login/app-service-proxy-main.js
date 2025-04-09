// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

document.addEventListener('DOMContentLoaded', function () {
  const Keycloak = window.Keycloak

  // Initialize Keycloak
  let domain = window.location.hostname.split('.').slice(1).join('.')
  if (domain === 'localhost') {
    // For local development
    domain = 'kind.internal'
  }
  const keycloak = new Keycloak({
    url: 'https://keycloak.' + domain,
    realm: 'master',
    clientId: 'webui-client'
  })

  const expectedParams = ['project', 'cluster', 'namespace', 'service']

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
      console.log(`Query parameter added as cookie: ${cookieName} = ${value}`)
      document.cookie = `${cookieName}=${value}; path=/; SameSite=Strict`
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

    // Redirect to the application
    const windowUrl = window.location.protocol + '//' + window.location.hostname + ':' + window.location.port
    console.log('Redirecting to: ' + windowUrl)

    // To facilitate local development, do not redirect if the URL contains 'localhost'
    if (!windowUrl.includes('localhost')) {
    //   window.location.href = windowUrl + '/'
      document.getElementById('asp-iframe').src = windowUrl
    }
  }).catch(function (err) {
    console.log('Failed to initialize', err)
  })
})
