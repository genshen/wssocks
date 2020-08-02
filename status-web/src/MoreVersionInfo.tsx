import React from 'react'

import { Text, Pane, UnorderedList, ListItem, PropertyIcon, Strong, Small, InfoSignIcon, DotIcon } from 'evergreen-ui'

function MoreVersionInfo() {
  return (
    <Pane
      width={300}
      height={128}
      padding="16px"
      display="flex"
      alignItems="center"
      justifyContent="center"
      flexDirection="column">
      <UnorderedList>
        <ListItem icon={PropertyIcon} iconColor="selected">
          <Text>Protocol Version:&nbsp;</Text>
          <Strong color="green">3</Strong>
        </ListItem>
        <ListItem display="flex" icon={PropertyIcon} iconColor="selected" 
          title="Compatible Protocol Version is the lowest protocal version allowed for client" >
          <Text>Compatible Protocol Version:&nbsp;</Text>
          <Strong color="green">3</Strong>
        </ListItem>
        <ListItem display="flex">
          <Small><DotIcon size={10} color="muted" /> <i>Compatible Protocol Version</i> is the lowest protocal version allowed for client.</Small>
        </ListItem>
      </UnorderedList>
    </Pane>
  )
}

export default MoreVersionInfo
