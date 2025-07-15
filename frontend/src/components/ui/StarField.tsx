'use client'

import { useEffect, useId, useRef } from 'react'
import clsx from 'clsx'

type Star = [x: number, y: number, dim?: boolean, blur?: boolean]

const stars: Array<Star> = [
  [4, 4, true, true],
  [4, 44, true],
  [36, 22],
  [50, 146, true, true],
  [64, 43, true, true],
  [76, 30, true],
  [101, 116],
  [140, 36, true],
  [149, 134],
  [162, 74, true],
  [171, 96, true, true],
  [210, 56, true, true],
  [235, 90],
  [275, 82, true, true],
  [306, 6],
  [307, 64, true, true],
  [380, 68, true],
  [380, 108, true, true],
  [391, 148, true, true],
  [405, 18, true],
  [412, 86, true, true],
  [426, 210, true, true],
  [427, 56, true, true],
  [538, 138],
  [563, 88, true, true],
  [611, 154, true, true],
  [637, 150],
  [651, 146, true],
  [682, 70, true, true],
  [683, 128],
  [781, 82, true, true],
  [785, 158, true],
  [832, 146, true, true],
  [852, 89],
]

const constellations: Array<Array<Star>> = [
  [
    [247, 103],
    [261, 86],
    [307, 104],
    [357, 36],
  ],
  [
    [586, 120],
    [516, 100],
    [491, 62],
    [440, 107],
    [477, 180],
    [516, 100],
  ],
  [
    [733, 100],
    [803, 120],
    [879, 113],
    [823, 164],
    [803, 120],
  ],
]

interface StarComponentProps {
  blurId: string
  point: Star
}

function Star({ blurId, point: [cx, cy, dim, blur] }: StarComponentProps) {
  let groupRef = useRef<React.ElementRef<'g'>>(null)
  let ref = useRef<React.ElementRef<'circle'>>(null)

  useEffect(() => {
    if (!groupRef.current || !ref.current) {
      return
    }

    let delay = Math.random() * 2

    // Simple CSS animation fallback for motion library
    const group = groupRef.current
    const circle = ref.current
    
    group.style.opacity = '0'
    
    setTimeout(() => {
      group.style.transition = 'opacity 4s ease-in-out'
      group.style.opacity = '1'
      
      circle.style.transition = `opacity ${Math.random() * 2 + 2}s ease-in-out infinite alternate, transform ${Math.random() * 2 + 2}s ease-in-out infinite alternate`
      circle.style.opacity = dim ? '0.2' : '1'
      circle.style.transform = `scale(${dim ? 1 : 1.2})`
    }, delay * 1000)

  }, [dim])

  return (
    <g ref={groupRef} className="opacity-0">
      <circle
        ref={ref}
        cx={cx}
        cy={cy}
        r={1}
        style={{
          transformOrigin: `${cx / 16}rem ${cy / 16}rem`,
          opacity: dim ? 0.2 : 1,
          transform: `scale(${dim ? 1 : 1.2})`,
        }}
        filter={blur ? `url(#${blurId})` : undefined}
      />
    </g>
  )
}

interface ConstellationProps {
  points: Array<Star>
  blurId: string
}

function Constellation({ points, blurId }: ConstellationProps) {
  let ref = useRef<React.ElementRef<'path'>>(null)
  let uniquePoints = points.filter(
    (point, pointIndex) =>
      points.findIndex((p) => String(p) === String(point)) === pointIndex,
  )
  let isFilled = uniquePoints.length !== points.length

  useEffect(() => {
    if (!ref.current) {
      return
    }

    const path = ref.current
    const delay = Math.random() * 3 + 2
    
    setTimeout(() => {
      path.style.transition = 'stroke-dashoffset 5s ease-in-out, visibility 0.1s'
      path.style.strokeDashoffset = '0'
      path.style.visibility = 'visible'
      
      if (isFilled) {
        setTimeout(() => {
          path.style.transition += ', fill 1s ease-in-out'
          path.style.fill = 'rgb(255 255 255 / 0.02)'
        }, 5000)
      }
    }, delay * 1000)

  }, [isFilled])

  return (
    <>
      <path
        ref={ref}
        stroke="white"
        strokeOpacity="0.2"
        strokeDasharray={1}
        strokeDashoffset={1}
        pathLength={1}
        fill="transparent"
        d={`M ${points.join('L')}`}
        className="invisible"
      />
      {uniquePoints.map((point, pointIndex) => (
        <Star key={pointIndex} point={point} blurId={blurId} />
      ))}
    </>
  )
}

interface StarFieldProps {
  className?: string
}

export function StarField({ className }: StarFieldProps) {
  let blurId = useId()

  return (
    <svg
      viewBox="0 0 881 211"
      fill="white"
      aria-hidden="true"
      className={clsx(
        'pointer-events-none absolute w-220.25 origin-top-right rotate-30 overflow-visible opacity-70',
        className,
      )}
    >
      <defs>
        <filter id={blurId}>
          <feGaussianBlur in="SourceGraphic" stdDeviation=".5" />
        </filter>
      </defs>
      {constellations.map((points, constellationIndex) => (
        <Constellation
          key={constellationIndex}
          points={points}
          blurId={blurId}
        />
      ))}
      {stars.map((point, pointIndex) => (
        <Star key={pointIndex} point={point} blurId={blurId} />
      ))}
    </svg>
  )
}