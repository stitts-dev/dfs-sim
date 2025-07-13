/**
 * Catalyst Link component integrated with React Router.
 * Uses React Router's Link component for client-side navigation.
 */

import * as Headless from '@headlessui/react'
import React, { forwardRef } from 'react'
import { Link as RouterLink } from 'react-router-dom'

export const Link = forwardRef(function Link(
  props: { href: string } & React.ComponentPropsWithoutRef<'a'>,
  ref: React.ForwardedRef<HTMLAnchorElement>
) {
  const { href, ...otherProps } = props
  
  // Check if it's an external link
  const isExternal = href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('tel:')
  
  if (isExternal) {
    return (
      <Headless.DataInteractive>
        <a {...props} ref={ref} />
      </Headless.DataInteractive>
    )
  }
  
  // Use React Router Link for internal navigation
  return (
    <Headless.DataInteractive>
      <RouterLink {...otherProps} to={href} ref={ref} />
    </Headless.DataInteractive>
  )
})
