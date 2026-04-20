import { openDB, type IDBPDatabase } from 'idb'

const DB_NAME = 'go-simple-chat-db'
const STORE_NAME = 'identity'
const DB_VERSION = 1

export const useIndexedDB = () => {
  let db: IDBPDatabase | null = null

  const getDB = async () => {
    if (db) return db
    db = await openDB(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          db.createObjectStore(STORE_NAME)
        }
      },
    })
    return db
  }

  const saveIdentity = async (cert: string, key: string) => {
    const db = await getDB()
    const tx = db.transaction(STORE_NAME, 'readwrite')
    const store = tx.objectStore(STORE_NAME)
    await Promise.all([
      store.put(cert, 'cert'),
      store.put(key, 'key'),
      tx.done
    ])
  }

  const loadIdentity = async (): Promise<{ cert: string, key: string } | null> => {
    try {
      const db = await getDB()
      const cert = await db.get(STORE_NAME, 'cert')
      const key = await db.get(STORE_NAME, 'key')
      
      if (cert && key) {
        return { cert, key }
      }
      return null
    } catch (e) {
      console.warn('Failed to load identity from IndexedDB:', e)
      return null
    }
  }

  const clearIdentity = async () => {
    const db = await getDB()
    const tx = db.transaction(STORE_NAME, 'readwrite')
    await tx.objectStore(STORE_NAME).clear()
    await tx.done
  }

  return {
    saveIdentity,
    loadIdentity,
    clearIdentity
  }
}
