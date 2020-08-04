import React, { useEffect, useState } from 'react';
import { Pane, Strong, UnorderedList, ListItem, Badge, Text, SwapHorizontalIcon } from 'evergreen-ui'
import { ChangesIcon, ApplicationsIcon, TickCircleIcon } from 'evergreen-ui'
import { Statistics } from './Status';

interface InfoBoardProps {
  data: Statistics
}

function upTimeString(t: number) {
  const seconds = t % 60
  t = Math.floor(t / 60) // left minutes
  const minutes = t % 60
  t = Math.floor(t / 60) // left hours
  const hours = t % 24
  const days = Math.floor(t / 24) // left days

  let re = seconds + ' second(s)'
  if (minutes === 0) {
    return re
  }
  re = minutes + ' minute(s) ' + re
  if (hours === 0) {
    return re
  }
  re = hours + ' hour(s) ' + re
  if (days === 0) {
    return re
  }
  return days + ' day(s) ' + re
}

function StatisticsBoard(props: InfoBoardProps) {
  const list_title_size = "150px"
  let [uptime, setUptime] = useState<number>(props.data.up_time)
  useEffect(() => {
    const timer = setInterval(() => {
      setUptime((uptime) => uptime + 1)
    }, 1000)
    return () => clearInterval(timer)
  }, [])

  return (
    <UnorderedList>
      <ListItem icon={TickCircleIcon} iconColor="success">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Status</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Badge color="green">In service</Badge>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={TickCircleIcon} iconColor="success">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Up Time</Text>
          </Pane>
          <Pane flexBasis={360}>
            <Strong color="green"> {upTimeString(uptime)} </Strong>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={ApplicationsIcon} iconColor="selected">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Clients</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green"> {props.data.clients} </Strong>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={SwapHorizontalIcon} iconColor="selected">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Proxy Connections</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green"> {props.data.proxies} </Strong>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={ChangesIcon} iconColor="selected">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Data Trans</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green"> unknown TiB</Strong>
          </Pane>
        </Pane>
      </ListItem>
    </UnorderedList>
  )
}

export default StatisticsBoard
