import React from 'react';
import { Pane, Button, Strong, UnorderedList, ListItem, Badge, Text, Heading, SwapHorizontalIcon } from 'evergreen-ui'
import { ChangesIcon, ApplicationsIcon, TickCircleIcon, BanCircleIcon, } from 'evergreen-ui'

function StatisticsBoard() {
  const list_title_size = "150px"
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
            <Text>Uptime (days)</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green">24</Strong>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={ApplicationsIcon} iconColor="selected">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Clients</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green">10</Strong>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={SwapHorizontalIcon} iconColor="selected">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Proxy Connections</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green">46</Strong>
          </Pane>
        </Pane>
      </ListItem>
      <ListItem icon={ChangesIcon} iconColor="selected">
        <Pane display="flex">
          <Pane minWidth={list_title_size}>
            <Text>Data Trans</Text>
          </Pane>
          <Pane flexBasis={120}>
            <Strong color="green">46 TiB</Strong>
          </Pane>
        </Pane>
      </ListItem>
    </UnorderedList>
  )
}

export default StatisticsBoard
