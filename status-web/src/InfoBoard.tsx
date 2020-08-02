import React from 'react'
import { Text, Pane, Popover, Position, Button, toaster, IconProps } from 'evergreen-ui'
import { DuplicateIcon, IconButton, KeyIcon, MoreIcon, LockIcon, DisableIcon, InfoSignIcon, TickCircleIcon } from 'evergreen-ui'
import { Row, Col } from 'react-grid-system'

import MoreVersionInfo from './MoreVersionInfo'
import { Info } from './Status'

type IconComponent = React.ForwardRefExoticComponent<React.PropsWithoutRef<Omit<IconProps, 'icon'>> & React.RefAttributes<SVGElement>>

interface InfoBoardProps {
  data: Info
}

interface StatusItemProps {
  title: string
  enable: boolean
  enableReason?: string
  disableReason: string
  enableIconColor?: string
  disableIconColor?: string
  enableTextColor?: string
  disableTextColor?: string
  enableIcon?: IconComponent
  disableIcon?: IconComponent
}


function StatusItem({
  title,
  enable,
  enableReason = "enabled",
  disableReason,
  enableIconColor = "success",
  disableIconColor = "disabled",
  enableTextColor = "success",
  disableTextColor = "muted",
}: StatusItemProps) {
  if (enable) {
    return (
      <>
        <Text margin="8px"> {title} </Text>
        <TickCircleIcon color={enableIconColor} />
        <Text color={enableTextColor} margin="8px"> {enableReason} </Text>
      </>
    )
  } else {
    return (
      <>
        <Text margin="8px"> {title} </Text>
        <DisableIcon color={disableIconColor} />
        <Text color={disableTextColor} margin="8px"> {disableReason} </Text>
      </>
    )
  }
}

function InfoBoard(props: InfoBoardProps) {
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
            <Text margin="8px"> Server Version</Text>
            <Text color="selected"> {props.data.version.version_str} </Text>
            <Popover content={
              <MoreVersionInfo
                version_code={props.data.version.version_code}
                compatible_version={props.data.version.compatible_version}
              />}>
              <IconButton marginLeft="8px" appearance="minimal" icon={MoreIcon} iconSize={18} />
            </Popover>
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px"> Address</Text>
            <Text color="selected"> ws://gensh.me </Text>
            <IconButton icon={DuplicateIcon} appearance="minimal" marginLeft="4px" onClick={() => onCopyAddr("ws://proxy.gensh.me")} >Minimal</IconButton>
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <StatusItem
              title="Socks5 Proxy"
              enable={props.data.socks5_enabled}
              disableReason={props.data.socks5_disabled_reason}
            />
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <StatusItem
              title="Http(s) Proxy"
              enable={props.data.http_enabled}
              disableReason={props.data.http_disabled_reason}
            />
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px"> Connction Key </Text>
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
              <KeyIcon color={props.data.conn_key_enable ? "success" : "disabled"} />
            </Popover>
            {props.data.conn_key_enable && (<>
              <Text color="success" margin="8px"> enabled </Text>
            </>)}
            {!props.data.conn_key_enable && (<>
              <Text color="muted" margin="8px"> {props.data.conn_key_disabled_reason} </Text>
            </>)}
          </Pane>
        </Col>
        <Col xs={12} sm={6} md={4} lg={3}>
          <Pane marginLeft="32px" display="flex" justifyContent="left" alignItems="center" flexDirection="row">
            <Text margin="8px"> SSL/TLS</Text>
            {props.data.ssl_enabled && (<>
              <LockIcon color="success" />
              <Text color="success" margin="8px"> enabled </Text>
            </>)}
            {!props.data.ssl_enabled && (<>
              <LockIcon color="disabled" />
              <Text color="muted" margin="8px"> {props.data.ssl_disabled_reason} </Text>
            </>)}
          </Pane>
        </Col>
      </Row>
    </Pane>
  )
}

export default InfoBoard
