import React, { useState } from 'react';

function App() {
  const [chatLog, setChatLog] = useState<{role: string, text: string}[]>([
    { role: 'ai', text: 'Halo! Saya asisten finansial AI UMKM Anda. Ada yang bisa saya bantu hari ini?' }
  ]);
  const [msg, setMsg] = useState('');

  const handleSend = () => {
    if (!msg.trim()) return;
    setChatLog([...chatLog, { role: 'user', text: msg }]);
    setMsg('');
    
    // Simulate AI thinking & reply
    setTimeout(() => {
      setChatLog(prev => [...prev, { 
        role: 'ai', 
        text: 'Mengambil data dari Accounting Server... Laba bersih Anda bulan ini adalah Rp 15.000.000. Performa bisnis sangat baik!' 
      }]);
    }, 1500);
  };

  const [staffForm, setStaffForm] = useState({ username: '', email: '', password: '', phoneNumber: '', role: 'kasir' });
  const [staffMsg, setStaffMsg] = useState('');

  const [productForm, setProductForm] = useState({ name: '', price: '', description: '' });
  const [productMsg, setProductMsg] = useState('');
  const [products, setProducts] = useState<any[]>([]);

  // Fetch products on load
  React.useEffect(() => {
    fetch('http://localhost:8201/products', {
      headers: { 'X-Tenant-ID': '19354900-b974-4e62-8dc8-85b3b93db3a4' }
    })
    .then(r => r.json())
    .then(data => { if (data.success) setProducts(data.data) })
    .catch(e => console.error(e));
  }, []);

  const handleAddProduct = async () => {
    setProductMsg('Menyimpan...');
    try {
      const res = await fetch('http://localhost:8201/products', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'X-Tenant-ID': '19354900-b974-4e62-8dc8-85b3b93db3a4'
        },
        body: JSON.stringify({
          name: productForm.name,
          price: parseFloat(productForm.price),
          description: productForm.description
        })
      });
      const data = await res.json();
      if (data.success) {
        setProductMsg('Produk ditambahkan!');
        setProducts([{ id: data.data.id, name: productForm.name, price: parseFloat(productForm.price), description: productForm.description }, ...products]);
        setProductForm({ name: '', price: '', description: '' });
      } else {
        setProductMsg(data.message || 'Gagal menambahkan produk.');
      }
    } catch (err) {
      setProductMsg('Terjadi kesalahan koneksi.');
    }
  };

  const handleAddStaff = async () => {
    setStaffMsg('Menyimpan...');
    try {
      const res = await fetch('http://localhost:8000/auth/add-staff', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'X-Tenant-ID': '19354900-b974-4e62-8dc8-85b3b93db3a4' // Mocking logged in tenant
        },
        body: JSON.stringify(staffForm)
      });
      const data = await res.json();
      if (data.success) {
        setStaffMsg('Pegawai berhasil ditambahkan!');
        setStaffForm({ username: '', email: '', password: '', phoneNumber: '', role: 'kasir' });
      } else {
        setStaffMsg(data.message || 'Gagal menambahkan pegawai.');
      }
    } catch (err) {
      setStaffMsg('Terjadi kesalahan koneksi.');
    }
  };

  return (
    <div className="container">
      <header className="header">
        <h1 className="title">WCH Enterprise Dashboard</h1>
        <button className="btn">Logout</button>
      </header>

      <div className="grid">
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          <div className="glass-card">
            <h2 style={{ marginBottom: '1.5rem', color: '#f8fafc' }}>Ringkasan Akuntansi</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', paddingBottom: '0.5rem', borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
                <span style={{ color: '#94a3b8' }}>Total Kas</span>
                <strong style={{ fontSize: '1.1rem' }}>Rp 45.000.000</strong>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', paddingBottom: '0.5rem', borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
                <span style={{ color: '#94a3b8' }}>Pendapatan</span>
                <strong style={{ color: '#4ade80', fontSize: '1.1rem' }}>Rp 20.000.000</strong>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ color: '#94a3b8' }}>Beban Operasional</span>
                <strong style={{ color: '#f87171', fontSize: '1.1rem' }}>Rp 5.000.000</strong>
              </div>
            </div>
          </div>

          <div className="glass-card">
            <h2 style={{ marginBottom: '1.5rem', color: '#f8fafc' }}>Pengaturan Pegawai (Kasir)</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              <input type="text" placeholder="Username" value={staffForm.username} onChange={e => setStaffForm({...staffForm, username: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <input type="email" placeholder="Email" value={staffForm.email} onChange={e => setStaffForm({...staffForm, email: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <input type="password" placeholder="Password" value={staffForm.password} onChange={e => setStaffForm({...staffForm, password: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <input type="text" placeholder="No WhatsApp (contoh: 0812...)" value={staffForm.phoneNumber} onChange={e => setStaffForm({...staffForm, phoneNumber: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <select value={staffForm.role} onChange={e => setStaffForm({...staffForm, role: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.5)', color: 'white' }}>
                <option value="kasir">Kasir</option>
                <option value="admin">Admin</option>
              </select>
              <button className="btn" onClick={handleAddStaff}>Tambah Pegawai</button>
              {staffMsg && <div style={{ marginTop: '0.5rem', color: '#4ade80' }}>{staffMsg}</div>}
            </div>
          </div>

          <div className="glass-card">
            <h2 style={{ marginBottom: '1.5rem', color: '#f8fafc' }}>Katalog Produk</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem', marginBottom: '1.5rem' }}>
              <input type="text" placeholder="Nama Produk" value={productForm.name} onChange={e => setProductForm({...productForm, name: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <input type="number" placeholder="Harga (misal: 15000)" value={productForm.price} onChange={e => setProductForm({...productForm, price: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <input type="text" placeholder="Deskripsi Singkat (opsional)" value={productForm.description} onChange={e => setProductForm({...productForm, description: e.target.value})} style={{ padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white' }} />
              <button className="btn" onClick={handleAddProduct} style={{ background: '#3b82f6' }}>Simpan Produk</button>
              {productMsg && <div style={{ color: '#4ade80' }}>{productMsg}</div>}
            </div>
            
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', maxHeight: '150px', overflowY: 'auto' }}>
              {products.map(p => (
                <div key={p.id} style={{ display: 'flex', justifyContent: 'space-between', padding: '0.5rem', borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
                  <span>{p.name}</span>
                  <strong style={{ color: '#4ade80' }}>Rp {p.price.toLocaleString()}</strong>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', height: '1100px' }}>
          <h2 style={{ marginBottom: '1rem', color: '#f8fafc' }}>Tanya AI Chatbot</h2>
          <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '1rem', paddingRight: '0.5rem' }}>
            {chatLog.map((chat, i) => (
              <div key={i} className={`chat-message ${chat.role === 'user' ? 'chat-user' : 'chat-ai'}`}>
                {chat.text}
              </div>
            ))}
          </div>
          <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1.5rem' }}>
            <input 
              type="text" 
              value={msg}
              onChange={(e) => setMsg(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSend()}
              placeholder="Ketik pertanyaan Anda..." 
              style={{ flex: 1, padding: '0.75rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'rgba(0,0,0,0.2)', color: 'white', outline: 'none' }}
            />
            <button className="btn" onClick={handleSend}>Kirim</button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
