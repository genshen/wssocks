import React from 'react'
import { Text, Pane, Popover, Position, Button, toaster } from 'evergreen-ui'
import { DuplicateIcon, IconButton, KeyIcon, MoreIcon, LockIcon, InfoSignIcon, TickCircleIcon } from 'evergreen-ui'
import { Row, Col } from 'react-grid-system'

import MoreVersionInfo from './MoreVersionInfo'

function InfoBoard() {
  const onCopyAddr = (addr: string) => {
    toaster.success(
      'Remote server address copied',
      {
        description: 'address: ' + addr
      }
    )
  }

  return (
    <Pane alignItems="center">
      {/* flex={1} display="flex" justifyContent="space-between" */}
      <Row>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px" textAlign="center"> Server Version</Text>
            <Text color="selected"> v0.4.0 </Text>
            <Popover content={MoreVersionInfo}>
              <IconButton marginLeft="8px" appearance="minimal" icon={MoreIcon} iconSize={18} />
            </Popover>
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px" textAlign="center"> Address</Text>
            <Text color="selected"> ws://gensh.me </Text>
            <IconButton icon={DuplicateIcon} appearance="minimal" marginLeft="4px" onClick={() => onCopyAddr("ws://proxy.gensh.me")} >Minimal</IconButton>
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px" textAlign="center"> Socks5 Proxy Enabled</Text>
            <TickCircleIcon color="success" />
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px" textAlign="center"> Http(s) Proxy Enabled</Text>
            <TickCircleIcon color="success" />
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px" textAlign="center"> Connction Key </Text>
            <Popover
              trigger="hover"
              position={Position.BOTTOM_LEFT}
              content={
                <Pane width={240}
                  height={64}
                  display="flex"
                  padding="8px"
                  alignItems="center"
                  justifyContent="center"
                  flexDirection="row"
                >
                  <InfoSignIcon size={30} color="info" marginRight={16} />
                  <Text>You can get connection key from admin of server</Text>
                </Pane>
              }
            >
              <KeyIcon color="disabled" />
            </Popover>
            <Text color="muted" margin="8px" textAlign="center"> disabled </Text>
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px" textAlign="center"> SSL/TLS</Text>
            <LockIcon color="disabled" />
            <Text color="muted" margin="8px" textAlign="center"> not support </Text>
          </Pane>
        </Col>
      </Row>
    </Pane>
  )
}

export default InfoBoard
