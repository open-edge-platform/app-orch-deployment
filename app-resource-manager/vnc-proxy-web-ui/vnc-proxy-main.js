// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

import './keycloak.min.js'
import RFB from './rfb.js'

function deleteAllCookies () {
  const cookies = document.cookie.split(';')
  for (const cookie of cookies) {
    const cookieName = cookie.split('=')[0].trim()
    document.cookie = `${cookieName}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 UTC; SameSite=Strict`
  }
  console.log('All cookies have been deleted.')
}

document.addEventListener('DOMContentLoaded', function () {
  // Initialize Keycloak
  let domain = window.location.hostname.split('.').slice(1).join('.')
  if (domain === 'localhost') {
    // For local development
    domain = 'kind.internal'
  }
  const Keycloak = window.Keycloak
  const keycloak = new Keycloak({
    url: 'https://keycloak.' + domain,
    realm: 'master',
    clientId: 'webui-client'
  })

  let rfb
  let desktopName

  const expectedParams = ['project', 'app', 'cluster', 'vm']

  function connectedToServer (e) {
    status('Connected to ' + desktopName)
  }

  // This function is called when we are disconnected
  function disconnectedFromServer (e) {
    if (e.detail.clean) {
      status('Disconnected')
    } else {
      status('Something went wrong, connection is closed')
    }
  }

  // When this function is called, the server requires
  // credentials to authenticate
  function credentialsAreRequired (e) {
    throw new Error('Credentials are required')
  }

  // When this function is called we have received
  // a desktop name from the server
  function updateDesktopName (e) {
    desktopName = e.detail.name
  }

  // Since most operating systems will catch Ctrl+Alt+Del
  // before they get a chance to be intercepted by the browser,
  // we provide a way to emulate this key sequence.
  function sendCtrlAltDel () {
    rfb.sendCtrlAltDel()
    return false
  }

  // Show a status text in the top bar
  function status (text) {
    document.getElementById('status').textContent = text
  }

  // This function extracts the value of one variable from the
  // query string. If the variable isn't defined in the URL
  // it returns the default value instead.
  function readQueryVariable (name, defaultValue) {
    // A URL with a query parameter can look like this:
    // https://www.example.com?myqueryparam=myvalue
    //
    // Note that we use location.href instead of location.search
    // because Firefox < 53 has a bug w.r.t location.search
    const re = new RegExp('.*[?&]' + name + '=([^&#]*)')
    const match = document.location.href.match(re)

    if (match) {
      // We have to decode the URL since want the cleartext value
      return decodeURIComponent(match[1])
    }

    return defaultValue
  }

  keycloak.onAuthLogout = function () {
    console.log('Logout event triggered. Deleting cookies')
    deleteAllCookies()
  }

  keycloak.init({
    onLoad: 'login-required'
  }).then(function (authenticated) {
    console.log(authenticated ? 'User is authenticated' : 'User is not authenticated')

    // Check for required query parameters
    const queryParams = new URLSearchParams(window.location.search)
    expectedParams.forEach((param) => {
      if (!queryParams.has(param)) {
        console.log(`Query parameter missing: ${param}`)
        throw new Error(`Query parameter missing: ${param}`)
      }
    })

    // Set the token as a cookie. If it is longer than 2k break it up into multiple cookies
    // We need a cookie since the VNC server can't handle Authorization headers.
    const token = keycloak.token
    const maxCookieLength = 2000
    const numCookies = Math.ceil(token.length / maxCookieLength)
    for (let i = 0; i < numCookies; i++) {
      const cookieValue = token.substring(i * maxCookieLength, (i + 1) * maxCookieLength)
      // document.cookie = `keycloak-token-${i}=${cookieValue}; path=/;`;
      document.cookie = `keycloak-token-${i}=${cookieValue}; path=/; secure; SameSite=Strict`
    }
    document.cookie = `keycloak-tokens=${numCookies}; path=/; SameSite=Strict`
    console.log(`Token saved as ${numCookies} cookies`)

    let vncAddress = 'wss://'
    if (window.location.hostname === 'localhost') {
      vncAddress = 'vnc.kind.internal'
    } else {
      vncAddress += window.location.hostname
    }
    vncAddress += '/vnc'
    vncAddress += '/' + queryParams.get('project')
    vncAddress += '/' + queryParams.get('app')
    vncAddress += '/' + queryParams.get('cluster')
    vncAddress += '/' + queryParams.get('vm')
    console.log('VNC address: ' + vncAddress)
    status('Connecting to ' + vncAddress)

    rfb = new RFB(document.getElementById('screen'), vncAddress)

    // Add listeners to important events from the RFB module
    rfb.addEventListener('connect', connectedToServer)
    rfb.addEventListener('disconnect', disconnectedFromServer)
    rfb.addEventListener('credentialsrequired', credentialsAreRequired)
    rfb.addEventListener('desktopname', updateDesktopName)

    // Set parameters that can be changed on an active connection
    rfb.viewOnly = readQueryVariable('view_only', false)
    rfb.scaleViewport = readQueryVariable('scale', false)

    document.getElementById('send-ctrl-alt-del-button')
      .onclick = sendCtrlAltDel
  }).catch(function (err) {
    console.log('Failed to initialize', err)
  })
})
